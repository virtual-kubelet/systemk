package systemd

import (
	"time"

	"github.com/miekg/vks/pkg/system"
	"github.com/miekg/vks/pkg/unit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func unitToPod(units map[string]*unit.UnitState) *corev1.Pod {
	podname := ""
	namespace := ""
	status := ""
	for k, v := range units {
		podname = k           // parse! namespace/podname
		namespace = "default" // parse from k
		status = v.ActiveState
	}
	println(namespace)
	// order of the map is random, need to sort.
	containers := toContainers(units)
	containerStatuses := toContainerStatuses(units)

	p := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        podname,   // get pod name from
			Namespace:   "default", // parse from u.Name
			ClusterName: "cluster.local",
			//			UID:               // parse from u.Name
			//			CreationTimestamp: // do we know?
		},
		Spec: corev1.PodSpec{
			NodeName:   system.Hostname(),
			Volumes:    []corev1.Volume{},
			Containers: containers,
		},

		// podstatus is the sum of all container statusses???
		Status: corev1.PodStatus{
			Phase:      activeStateToPhase(status),
			Conditions: activeStateToPodConditions(status, metav1.NewTime(time.Now())),
			Message:    "",
			Reason:     "",
			HostIP:     "",
			PodIP:      "127.0.0.1",
			//			StartTime:         &containerStartTime,
			ContainerStatuses: containerStatuses,
		},
	}
	return p
}

func toContainers(units map[string]*unit.UnitState) []corev1.Container {
	// name
	return nil
}

func toContainerStatuses(units map[string]*unit.UnitState) []corev1.ContainerStatus {
	//	for name, unit := range units {
	//
	//	}
	return nil
}

/*
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

	return &p, nil
}
*/
