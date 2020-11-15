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
func (p *P) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) { return nil, nil }

// GetPods returns a list of all pods known
//func (p *P) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
//	// multiple containers are multiple units, we need get those back into a single pod
//	return nil, nil
//}

func (p *P) GetPods(_ context.Context) ([]*corev1.Pod, error) {
	_, err := p.m.ListUnits()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *P) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	println("gET PODS")
	fmt.Printf("+%v\n", pod)
	return nil

}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *P) RunInContainer(ctx context.Context, namespace, name, container string, cmd []string, attach api.AttachIO) error {
	// not implemented, because we can't
	log.Printf("receive ExecInContainer %q\n", container)
	return nil
}

// GetPodStatus returns the status of a pod by name that is running inside Zun
// returns nil if a pod by that name is not found.
func (p *P) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	return nil, nil
}

func (p *P) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	// systemd
	return ioutil.NopCloser(strings.NewReader("not support in systemd provider")), nil
}

/* keep because of how to create corev1.Pod object
func capsuleToPod(capsule *capsules.CapsuleV132) (*v1.Pod, error) {
	var podCreationTimestamp metav1.Time
	var containerStartTime metav1.Time

	podCreationTimestamp = metav1.NewTime(capsule.CreatedAt)
	if len(capsule.Containers) > 0 {
		containerStartTime = metav1.NewTime(capsule.Containers[0].StartedAt)
	}
	containerStartTime = metav1.NewTime(time.Time{})
	// Deal with container inside capsule
	containers := make([]v1.Container, 0, len(capsule.Containers))
	containerStatuses := make([]v1.ContainerStatus, 0, len(capsule.Containers))
	for _, c := range capsule.Containers {
		containerMemoryMB := 0
		if c.Memory != "" {
			containerMemory, err := strconv.Atoi(c.Memory)
			if err != nil {
				log.Println(err)
			}
			containerMemoryMB = containerMemory
		}
		container := v1.Container{
			Name:    c.Name,
			Image:   c.Image,
			Command: c.Command,
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%g", float64(c.CPU))),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dM", containerMemoryMB)),
				},

				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%g", float64(c.CPU*1024/100))),
					v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dM", containerMemoryMB)),
				},
			},
		}
		containers = append(containers, container)
		containerStatus := v1.ContainerStatus{
			Name:                 c.Name,
			State:                zunContainerStausToContainerStatus(&c),
			LastTerminationState: zunContainerStausToContainerStatus(&c),
			Ready:                zunStatusToPodPhase(c.Status) == v1.PodRunning,
			RestartCount:         int32(0),
			Image:                c.Image,
			ImageID:              "",
			ContainerID:          c.UUID,
		}

		// Add to containerStatuses
		containerStatuses = append(containerStatuses, containerStatus)
	}

	ip := ""
	if capsule.Addresses != nil {
		for _, v := range capsule.Addresses {
			for _, addr := range v {
				if addr.Version == float64(4) {
					ip = addr.Addr
				}
			}
		}
	}
	p := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              capsule.MetaLabels["PodName"],
			Namespace:         capsule.MetaLabels["Namespace"],
			ClusterName:       capsule.MetaLabels["ClusterName"],
			UID:               types.UID(capsule.UUID),
			CreationTimestamp: podCreationTimestamp,
		},
		Spec: v1.PodSpec{
			NodeName:   capsule.MetaLabels["NodeName"],
			Volumes:    []v1.Volume{},
			Containers: containers,
		},

		Status: v1.PodStatus{
			Phase:             zunStatusToPodPhase(capsule.Status),
			Conditions:        zunStatusToPodConditions(capsule.Status, podCreationTimestamp),
			Message:           "",
			Reason:            "",
			HostIP:            "",
			PodIP:             ip,
			StartTime:         &containerStartTime,
			ContainerStatuses: containerStatuses,
		},
	}

	return &p, nil
}
*/

// UpdatePod is a noop,
func (p *P) UpdatePod(ctx context.Context, pod *corev1.Pod) error { return nil }

// DeletePod deletes
func (p *P) DeletePod(ctx context.Context, pod *corev1.Pod) error { return nil }

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
