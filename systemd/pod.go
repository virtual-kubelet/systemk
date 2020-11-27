package systemd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path"
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
		return err
	}
	if len(pod.Spec.Containers) > 1 {
		return fmt.Errorf("more than 1 (%d) containers isn't tested/supported yet", len(pod.Spec.Containers))
	}
	for _, c := range pod.Spec.Containers {
		tmp := []string{"/var", "/run"}
		bindmounts := []string{} // per pod
		for _, v := range c.VolumeMounts {
			// readonly needs to be handled here. We need slighty more than a map and something to turn it into systemd directives.
			dir, ok := vol[v.Name]
			if !ok {
				log.Printf("failed to find volumeMount %s in the specific volumes, skpping", v.Name)
				continue
			}
			tmp = append(tmp, v.MountPath)
			if dir == "" { // empty dir emptyDir, no bind mount for this one
				continue
			}

			bindmounts = append(bindmounts, fmt.Sprintf("%s:%s", dir, v.MountPath)) // SubPath, look at todo, filepath.Join?
		}

		// TODO(): parse c.Image for tag to get version
		err, installed := p.pkg.Install(c.Image, "")
		if err != nil {
			log.Printf("Failed to install: %s", err)
			return err
		}

		uf, err := p.unitfileFromPackageOrSynthesized(c, installed)
		if err != nil {
			log.Printf("Failed to create/use unit file: %s", err)
			return err
		}

		// do permissions properly, TODO(miek): fix soon!! We need to chmod our mountpoints (sadly)
		uf = uf.Overwrite("Service", "User", "root")
		// keep the unit around, the control plane where clear it with a DeletePod
		uf = uf.Overwrite("Service", "RemainAfterExit", "true")

		// If command and/or args are given we need to override the ExecStart
		// Bit execStart should be a string slice, but isn't returned like this, so this involves some string wrangling
		// to get things right.
		execStart := uf.Contents["Service"]["ExecStart"] // check if exists...?
		execStarts := []string(execStart)
		if len(execStart) == 1 {
			execStarts = strings.Fields(execStart[0])
		}

		if c.Command != nil {
			if cmd := c.Command[0]; !path.IsAbs(cmd) {
				fullpath, err := exec.LookPath(cmd)
				if err == nil {
					c.Command[0] = fullpath
				}
				// if this errored the unit will fail, so fail there instead of erroring here.
			}
			execStarts[0] = strings.Join(c.Command, " ") // some args might be included here
		}
		if c.Args != nil {
			execStarts = execStarts[:1]
			execStarts = append(execStarts, c.Args...)
		}
		uf = uf.Overwrite("Service", "ExecStart", strings.Join(execStarts, " "))

		name := PodToUnitName(pod, c.Name)

		uid := string(pod.ObjectMeta.UID) // give multiple containers the same access? Need to test this.
		uf = uf.Insert(kubernetesSection, "namespace", pod.ObjectMeta.Namespace)
		uf = uf.Insert(kubernetesSection, "clusterName", pod.ObjectMeta.ClusterName)
		uf = uf.Insert(kubernetesSection, "uid", uid)
		tmpfs := strings.Join(tmp, " ")
		uf = uf.Insert("Service", "TemporaryFileSystem", tmpfs)
		mount := strings.Join(bindmounts, " ")
		uf = uf.Insert("Service", "BindPaths", mount)

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
