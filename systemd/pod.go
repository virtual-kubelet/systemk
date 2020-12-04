package systemd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/miekg/vks/pkg/unit"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
)

// If any of these methods return an error, it will show up in the kubectl output as "ProviderFailed", so we should
// be very careful to just return one of something trivial failed. It's better to setup as much as you can then let
// the container/unit start fail, which will be correctly picked up by the control plane.

func (p *P) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	log.Print("GetPod called")
	stats, err := p.m.States(Prefix)
	if err != nil {
		log.Printf("Failed to get states: %s", err)
		return nil, nil
	}
	pod := p.statsToPod(stats)
	return pod, nil
}

func (p *P) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	states, err := p.m.States(Prefix)
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, nil
	}

	// Get all the names and then we just call GetPod for each.
	ns := map[string][]string{} // namespace/ pod(s) mapping

	// sort unit by namespace/name
	for name := range states {
		namespace := Namespace(name)
		pod := Pod(name)
		ns[namespace] = append(ns[namespace], pod)
	}

	pods := []*corev1.Pod{}
	for namespace, names := range ns {
		for _, name := range names {
			if pod, err := p.GetPod(ctx, namespace, name); err != nil {
				pods = append(pods, pod)
			}
		}
	}
	return pods, nil
}

func (p *P) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Print("CreatedPod called")

	vol, err := p.volumes(pod)
	if err != nil {
		log.Printf("Failed to setup volumes: %s", err)
	}

	uid, gid := UidGidFromSecurityContext(pod)
	tmp := []string{"/var", "/run"}

	for _, c := range pod.Spec.Containers {
		bindmounts := []string{}
		bindmountsro := []string{}
		rwpaths := []string{} // everything is RO, this will enable to pod to write to specific dirs.
		for _, v := range c.VolumeMounts {
			dir, ok := vol[v.Name]
			if !ok {
				log.Printf("failed to find volumeMount %s in the specific volumes, skpping", v.Name)
				continue
			}

			if v.ReadOnly {
				bindmountsro = append(bindmountsro, fmt.Sprintf("%s:%s", dir, v.MountPath)) // SubPath, look at todo, filepath.Join?
				continue
			}
			rwpaths = append(rwpaths, v.MountPath)
			if dir == "" { // hostPath
				continue
			}
			bindmounts = append(bindmounts, fmt.Sprintf("%s:%s", dir, v.MountPath)) // SubPath, look at todo, filepath.Join?

			// OK, so the v.MountPath will _exist_ on the system, as systemd will create it, BUT the permissions/ownership might be wrong
			// We will chown the directory to the user/group of the security context, but the directory is created by systemd, when it
			// performs the automount. We'll add ExecPreStart that chowns the directory, but only if it's empty, otherwise we might be
			// messing with some system directory - this may still be the case when the dir is empty though. Because a StartExecPre to check
			// this would be a shell script we make the check here:
			// * create/chown if not exist
			// * chown if empty dir
			// * anything else don't touch: this likely makes the unit start, but then fail. (we may intercept this and return a decent message
			//   so you can see this with events
			//
			// Cleanup of these directories is hard - systemd clean <unit> exists for doesn't assume other units are using this space.
			_, err := os.Stat(v.MountPath)
			switch os.IsNotExist(err) {
			case true: // doesn't exist
				log.Printf("Creating %q", v.MountPath)
				if err := os.MkdirAll(v.MountPath, dirPerms); err != nil {
					return err // return err? This will be an event. TODO: tweak message
				}
				log.Printf("Chowning %q", v.MountPath)
				if err := chown(v.MountPath, uid, gid); err != nil {
					return err
				}
			case false: // exist
				empty, err := isEmpty(v.MountPath)
				if err != nil {
					return err
				}
				if !empty {
					log.Printf("Directory %q is not empty, refusing to touch", v.MountPath)
					// error, log, something??
					break
				}
				log.Printf("Chowning %q", v.MountPath)
				if err := chown(v.MountPath, uid, gid); err != nil {
					return err
				}
			}
		}

		// TODO(): parse c.Image for tag to get version. Check ImagePullAways to reinstall??
		// if we're downloading the image, the image name needs cleaning
		err, installed := p.pkg.Install(c.Image, "")
		if err != nil {
			log.Printf("Failed to install package %q: %s", c.Image, err)
			return err
		}

		c.Image = p.pkg.Clean(c.Image) // clean up the image if fetched with https

		uf, err := p.unitfileFromPackageOrSynthesized(c, installed)
		if err != nil {
			log.Printf("Failed to create/use unit file for %q: %s", c.Image, err)
			return err
		}
		// disable the systemd unit for the system if there, and only if we installed the package
		if installed {
			unitfile, err := p.pkg.Unitfile(c.Image)
			if err != nil {
				break
			}
			if err := p.m.Mask(unitfile); err != nil {
				log.Printf("failed to mask system unit %q: %s", unitfile, err)
			}
		}

		uf = uf.Overwrite("Service", "ProtectSystem", "true")
		uf = uf.Overwrite("Service", "ProtectHome", "tmpfs")
		uf = uf.Overwrite("Service", "PrivateMounts", "true")
		uf = uf.Overwrite("Service", "ReadOnlyPaths", "/")
		uf = uf.Insert("Service", "StandardOutput", "journal")
		uf = uf.Insert("Service", "StandardError", "journal")

		// What do we do with the defaults from the unit file - they are probably more sensible than blindly running as root.
		// Keep them? TODO(miek): needs to be documented.
		// If there is a securityContext we'll use that.
		if uid != "" {
			uf = uf.Overwrite("Service", "User", uid)
		}
		if gid != "" {
			uf = uf.Overwrite("Service", "Group", gid)
		}

		// keep the unit around, the control plane where clear it with a DeletePod.
		// this is also for us to return the state even after the unit left the stage.
		uf = uf.Overwrite("Service", "RemainAfterExit", "true")

		execStart := commandAndArgs(uf, c)
		uf = uf.Overwrite("Service", "ExecStart", strings.Join(execStart, " "))

		name := PodToUnitName(pod, c.Name)

		id := string(pod.ObjectMeta.UID) // give multiple containers the same access? Need to test this.
		uf = uf.Insert(kubernetesSection, "Namespace", pod.ObjectMeta.Namespace)
		uf = uf.Insert(kubernetesSection, "ClusterName", pod.ObjectMeta.ClusterName)
		uf = uf.Insert(kubernetesSection, "Id", id)

		tmpfs := strings.Join(tmp, " ")
		uf = uf.Insert("Service", "TemporaryFileSystem", tmpfs)
		if len(rwpaths) > 0 {
			paths := strings.Join(rwpaths, " ")
			uf = uf.Insert("Service", "ReadWritePaths", paths)
		}
		if len(bindmounts) > 0 {
			mount := strings.Join(bindmounts, " ")
			uf = uf.Insert("Service", "BindPaths", mount)
		}
		if len(bindmountsro) > 0 {
			romount := strings.Join(bindmountsro, " ")
			uf = uf.Insert("Service", "BindReadOnlyPaths", romount)
		}

		for _, env := range p.defaultEnvironment() {
			uf = uf.Insert("Service", "Environment", env)
		}

		log.Printf("Starting unit %s, %s as %s\n%s", c.Name, c.Image, name, uf)
		if err := p.m.Load(name, *uf); err != nil {
			log.Printf("Failed to load unit: %s", err)
			return err
		}
		if err := p.m.TriggerStart(name); err != nil {
			log.Printf("Failed to trigger start: %s", err)
			return err
		}
	}
	return nil
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *P) RunInContainer(ctx context.Context, namespace, name, container string, cmd []string, attach api.AttachIO) error {
	// Should we just try to start something? But with what user???
	log.Printf("receive RunInContainer %q\n", container)
	return nil
}

// GetPodStatus returns the status of a pod by name that is running.
// returns nil if a pod by that name is not found.
func (p *P) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	log.Printf("GetPodStatus called")
	pod, err := p.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if pod == nil {
		return nil, nil
	}
	return &pod.Status, nil
}

func (p *P) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	log.Printf("GetContainerLogs called")

	unitname := UnitPrefix(namespace, podName) + separator + containerName
	args := []string{"-u", unitname}
	cmd := exec.Command("journalctl", args...)
	// returns the buffers? What about following, use pipes here or something?
	buf, err := cmd.CombinedOutput()
	return ioutil.NopCloser(bytes.NewReader(buf)), err
}

// UpdatePod is a noop,
func (p *P) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("UpdatePod called - not implemented")
	return nil
}

// DeletePod deletes a pod.
func (p *P) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("DeletePod called")
	for _, c := range pod.Spec.Containers {
		name := PodToUnitName(pod, c.Name)
		if err := p.m.TriggerStop(name); err != nil {
			log.Printf("Failed to triggger top: %s", err)
		}
		if err := p.m.Unload(name); err != nil {
			log.Printf("Failed to unload: %s", err)
		}
	}
	p.m.Reload()
	return nil
}

func PodToUnitName(pod *corev1.Pod, containerName string) string {
	return UnitPrefix(pod.Namespace, pod.Name) + separator + containerName + unit.ServiceSuffix
}

func UnitPrefix(namespace, podName string) string {
	return Prefix + separator + namespace + separator + podName
}

func Image(name string) string {
	el := strings.Split(name, separator) // assume well formed
	if len(el) < 4 {
		return ""
	}
	return el[3]
}

func Name(name string) string {
	el := strings.Split(name, separator) // assume well formed
	if len(el) < 4 {
		return ""
	}
	return el[1] + separator + el[2]
}

func Pod(name string) string {
	el := strings.Split(name, separator) // assume well formed
	if len(el) < 4 {
		return ""
	}
	return el[2]
}

func Namespace(name string) string {
	el := strings.Split(name, separator) // assume well formed
	if len(el) < 4 {
		return ""
	}
	return el[1]
}

const (
	// Prefix the unit file prefix we used.
	Prefix    = "vks"
	separator = "."
)
