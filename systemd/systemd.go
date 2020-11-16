package systemd

// copied from virtual kubelet zun

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
)

// GetPod returns ...
func (p *P) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	println("GET POD")
	return nil, nil
}

func (p *P) GetPods(_ context.Context) ([]*corev1.Pod, error) {
	// focus on starting a unit first
	return nil, nil
	u, err := p.m.ListUnits()
	if err != nil {
		return nil, err
	}
	pods := make([]*corev1.Pod, len(u))
	for i := range u {
		pods[i] = unitToPod(u[i])
	}

	return pods, nil
}

func (p *P) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	println("CREATE PODS")
	fmt.Printf("+%v\n", pod)
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

// GetPodStatus returns the status of a pod by name that is running inside Zun
// returns nil if a pod by that name is not found.
func (p *P) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	println("GET POD STATUS")
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

/*
func zunContainerStausToContainerStatus(cs *capsules.Container) v1.ContainerState {
	// Zun already container start time but not add support at gophercloud
	//startTime := metav1.NewTime(time.Time(cs.StartTime))

	// Zun container status:
	//'Error', 'Running', 'Stopped', 'Paused', 'Unknown', 'Creating', 'Created',
	//'Deleted', 'Deleting', 'Rebuilding', 'Dead', 'Restarting'

	// Handle the case where the container is running.
	if cs.Status == "Running" || cs.Status == "Stopped" {
		return v1.ContainerState{
			Running: &v1.ContainerStateRunning{
				StartedAt: metav1.NewTime(time.Time(cs.StartedAt)),
			},
		}
	}

	// Handle the case where the container failed.
	if cs.Status == "Error" || cs.Status == "Dead" {
		return v1.ContainerState{
			Terminated: &v1.ContainerStateTerminated{
				ExitCode:   int32(0),
				Reason:     cs.Status,
				Message:    cs.StatusDetail,
				StartedAt:  metav1.NewTime(time.Time(cs.StartedAt)),
				FinishedAt: metav1.NewTime(time.Time(cs.UpdatedAt)),
			},
		}
	}

	// Handle the case where the container is pending.
	// Which should be all other Zun states.
	return v1.ContainerState{
		Waiting: &v1.ContainerStateWaiting{
			Reason:  cs.Status,
			Message: cs.StatusDetail,
		},
	}
}

func zunStatusToPodPhase(status string) v1.PodPhase {
	switch status {
	case "Running":
		return v1.PodRunning
	case "Stopped":
		return v1.PodSucceeded
	case "Error":
		return v1.PodFailed
	case "Dead":
		return v1.PodFailed
	case "Creating":
		return v1.PodPending
	case "Created":
		return v1.PodPending
	case "Restarting":
		return v1.PodPending
	case "Rebuilding":
		return v1.PodPending
	case "Paused":
		return v1.PodPending
	case "Deleting":
		return v1.PodPending
	case "Deleted":
		return v1.PodPending
	}

	return v1.PodUnknown
}

func zunStatusToPodConditions(status string, transitiontime metav1.Time) []v1.PodCondition {
	switch status {
	case "Running":
		return []v1.PodCondition{
			v1.PodCondition{
				Type:               v1.PodReady,
				Status:             v1.ConditionTrue,
				LastTransitionTime: transitiontime,
			}, v1.PodCondition{
				Type:               v1.PodInitialized,
				Status:             v1.ConditionTrue,
				LastTransitionTime: transitiontime,
			}, v1.PodCondition{
				Type:               v1.PodScheduled,
				Status:             v1.ConditionTrue,
				LastTransitionTime: transitiontime,
			},
		}
	}
	return []v1.PodCondition{}
}
*/

// IsVirtuelKubeletUnit returns true of the name of the unit is managed by virtual kubelet. Right now
// this means it's a `.service` the name starts with `vks-`.
func IsVirtuelKubeletUnit(name string) bool {
	if strings.HasPrefix(name, "vks-") {
		return true
	}
	if strings.HasSuffix(name, ".service") {
		return true
	}
	return false
}

func ImageNameToUnitName(name string) string {
	return "vks-" + name + ".service"
}
