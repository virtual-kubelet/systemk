package systemd

// copied from virtual kubelet zun

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/miekg/vks/pkg/unit"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
)

// GetPod returns ...
func (p *P) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	log.Print("GetPod called")
	units, err := p.m.GetUnitStates(Prefix)
	if err != nil {
		return nil, err
	}
	unitpref := UnitPrefix(namespace, name)
	for name, _ := range units {
		if !strings.HasPrefix(name, unitpref) {
			delete(units, name)
		}
	}
	pod := unitToPod(units)
	return pod, nil
}

func (p *P) GetPods(_ context.Context) ([]*corev1.Pod, error) {
	states, err := p.m.GetUnitStates(Prefix)
	if err != nil {
		return nil, err
	}
	// sort unit by namespace/name
	for name, s := range states {
		fmt.Printf("GETPODS: %s: %+v\n", name, s)
	}
	return nil, nil
}

func (p *P) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Print("CreatedPod called")
	// Can we store metadata somewhere within systemd units files?
	/*
		metadata.Labels = map[string]string{
			"PodName":           pod.Name,
			"ClusterName":       pod.ClusterName,
			"NodeName":          pod.Spec.NodeName,
			"Namespace":         pod.Namespace,
			"UID":               podUID,
			"CreationTimestamp": podCreationTimestamp,
		}
	*/
	for _, c := range pod.Spec.Containers {
		// TODO(miek): parse c.Image for tag to get version
		if err := p.pkg.Install(c.Image, ""); err != nil {
			return err
		}
		u, err := p.pkg.Unitfile(c.Image)
		if err != nil {
			return err
		}
		name := PodToUnitName(pod, c.Name)
		log.Printf("Starting unit %s, %s as %s", c.Name, c.Image, name)
		buf, err := ioutil.ReadFile(u)
		if err != nil {
			return err
		}

		// Inject all the metadata into it
		meta := objectMetaToSection(pod.ObjectMeta)
		buf = append(buf, meta...)

		uf, err := unit.NewUnitFile(string(buf))
		if err != nil {
			return err
		}
		if err := p.m.Load(name, *uf); err != nil {
			return err
		}
		if err := p.m.TriggerStart(name); err != nil {
			return err
		}

	}
	return nil
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *P) RunInContainer(ctx context.Context, namespace, name, container string, cmd []string, attach api.AttachIO) error {
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
	return ioutil.NopCloser(strings.NewReader("not support in systemd provider")), nil
}

// UpdatePod is a noop,
func (p *P) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("UpdatePod called")
	return nil
}

// DeletePod deletes
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

func PodToUnitName(pod *corev1.Pod, image string) string {
	return UnitPrefix(pod.Namespace, pod.Name) + Separator + image + ".service"
}

func UnitPrefix(namespace, name string) string {
	return Prefix + Separator + namespace + Separator + name
}

func Image(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 3 {
		return ""
	}
	return el[2]
}

func Name(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 3 {
		return ""
	}
	return el[1] + Separator + el[2]
}

func Pod(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 3 {
		return ""
	}
	return el[2]
}

func Namespace(name string) string {
	el := strings.Split(name, Separator) // assume well formed
	if len(el) < 3 {
		return ""
	}
	return el[1]
}

const (
	Prefix    = "vks"
	Separator = "."
)
