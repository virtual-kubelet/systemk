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
	states, err := p.m.GetUnitStates(Prefix)
	if err != nil {
		return nil, err
	}
	unitpref := UnitPrefix(namespace, name)
	for name, s := range states {
		if strings.HasPrefix(name, unitpref) {
			fmt.Printf("GETPOD: %+v\n", s)
		}
	}
	return nil, nil
}

func (p *P) GetPods(_ context.Context) ([]*corev1.Pod, error) {
	states, err := p.m.GetUnitStates(Prefix)
	if err != nil {
		return nil, err
	}
	for name, s := range states {
		fmt.Printf("GETPODS: %s: %+v\n", name, s)
	}
	return nil, nil
}

func (p *P) CreatePod(ctx context.Context, pod *corev1.Pod) error {
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
		metadata.Name = pod.Namespace + "-" + pod.Name
	*/
	// TODO: check if we have this pod (namespace/name??) already
	uid := string(pod.UID)
	for _, c := range pod.Spec.Containers {
		// parse c.Image for tag
		if err := p.Pkg.Install(c.Image, ""); err != nil {
			return err
		}
		u, err := p.Pkg.Unitfile(c.Image)
		if err != nil {
			return err
		}
		name := PodToUnitName(pod.Namespace, pod.Name, c.Name, uid)
		log.Printf("Starting unit %s, %s as %s", c.Name, c.Image, name)
		buf, err := ioutil.ReadFile(u)
		if err != nil {
			return err
		}
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
	println("RUN IN CONTAINER")
	// not implemented, because we can't
	log.Printf("receive ExecInContainer %q\n", container)
	return nil
}

// GetPodStatus returns the status of a pod by name that is running.
// returns nil if a pod by that name is not found.
func (p *P) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	states, err := p.m.GetUnitStates(Prefix)
	if err != nil {
		return nil, err
	}
	unitpref := UnitPrefix(namespace, name)
	for name, s := range states {
		if strings.HasPrefix(name, unitpref) {
			fmt.Printf("GETPODSTATUS: %+v\n", s)
		}
	}
	return nil, nil
}

func (p *P) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	println("GET CONTAINER LOGS")
	return ioutil.NopCloser(strings.NewReader("not support in systemd provider")), nil
}

// UpdatePod is a noop,
func (p *P) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	println("UPDATE POD")
	return nil
}

// DeletePod deletes
func (p *P) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	println("DELETE POD", pod.ObjectMeta.Name)
	return nil
}

func PodToUnitName(namespace, name, image, uid string) string {
	return UnitPrefix(namespace, name) + Separator + image + Separator + uid + ".service"
}

func UnitPrefix(namespace, name string) string {
	return Prefix + Separator + namespace + Separator + name
}

func UidFromUnitName(name string) string {
	// maybe we should hide this and put into some kind of systemd metadata...
	uid := name[:len(name)-len(".service")]
	sep := strings.LastIndex(uid, Separator)
	if sep == 0 {
		return ""
	}
	return uid[sep+1:]
}

const (
	Prefix    = "vks"
	Separator = "."
)
