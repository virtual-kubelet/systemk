package systemd

import (
	"github.com/miekg/go-systemd/dbus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func unitToPod(u dbus.UnitStatus) *corev1.Pod {
	/*
		type UnitStatus struct {
			Name        string          // The primary unit name as string
			Description string          // The human readable description string
			LoadState   string          // The load state (i.e. whether the unit file has been loaded successfully)
			ActiveState string          // The active state (i.e. whether the unit is currently started or not)
			SubState    string          // The sub state (a more fine-grained version of the active state that is specific to the unit type, which the active state is not)
			Followed    string          // A unit that is being followed in its state by this unit, if there is any, otherwise the empty string.
			Path        dbus.ObjectPath // The unit object path
			JobId       uint32          // If there is a job queued for the job unit the numeric job id, 0 otherwise
			JobType     string          // The job type as string
			JobPath     dbus.ObjectPath // The job object path
		}
	*/
	p := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        u.Name,
			Namespace:   "default",
			ClusterName: "cluster.local",
			//			UID:               types.UID(capsule.UUID),
			//			CreationTimestamp: podCreationTimestamp,
		},
		Spec: corev1.PodSpec{
			NodeName: hostname(),
			Volumes:  []corev1.Volume{},
			//Containers: containers,
		},

		Status: corev1.PodStatus{
			//Phase:      zunStatusToPodPhase(capsule.Status),
			///Conditions: zunStatusToPodConditions(capsule.Status, podCreationTimestamp),
			Message: "",
			Reason:  "",
			HostIP:  "",
			PodIP:   "127.0.0.1",
			//			StartTime:         &containerStartTime,
			//			ContainerStatuses: containerStatuses,
		},
	}
	return p
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
