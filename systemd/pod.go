package systemd

// copied from virtual kubelet zun

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

	"github.com/miekg/vks/pkg/unit"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
)

// GetPod returns ...
func (p *P) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	log.Print("GetPod called")
	units, err := p.m.GetStates(Prefix)
	if err != nil {
		return nil, err
	}
	unitpref := UnitPrefix(namespace, name)
	for name := range units {
		if !strings.HasPrefix(name, unitpref) {
			delete(units, name)
		}
	}
	pod := unitToPod(units)
	return pod, nil
}

func (p *P) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	states, err := p.m.GetStates(Prefix)
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
	for _, c := range pod.Spec.Containers {
		// TODO(miek): parse c.Image for tag to get version
		if err := p.pkg.Install(c.Image, ""); err != nil {
			return err
		}
		u, err := p.pkg.Unitfile(c.Image)
		if err != nil {
			return err
		}
		log.Printf("Unit file found at %q", u)
		name := PodToUnitName(pod, c.Name)
		log.Printf("Starting unit %s, %s as %s", c.Name, c.Image, name)
		buf, err := ioutil.ReadFile(u)
		if err != nil {
			return err
		}

		// Inject all the metadata into it.
		meta := objectMetaToSection(pod.ObjectMeta)
		buf = append(buf, meta...)

		uf, err := unit.New(string(buf))
		if err != nil {
			log.Printf("Failed to unit.New: %s", err)
			return err
		}
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
