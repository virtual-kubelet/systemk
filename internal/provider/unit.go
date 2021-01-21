package provider

import (
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/virtual-kubelet/systemk/internal/unit"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (p *p) statsToPod(stats map[string]*unit.State) *corev1.Pod {
	if len(stats) == 0 {
		return nil
	}
	name := ""
	// Pick a random unit, the things we care about should be identical between them.
	// We might need to pick the "correct" one at some point.
	for k := range stats {
		name = k
		break
	}
	uf, err := unit.NewFile(stats[name].UnitData)
	if err != nil {
		log.Error("error while parsing unit file: %s", err)
	}

	// If for some unknown reason we find a unit without kubernetesSection
	// proceed to remove it.
	if _, ok := uf.Contents[kubernetesSection]; !ok {
		log.Warnf("unit %q did not contain %s section, removing", name, kubernetesSection)
		// delete it
		if err := p.unitManager.TriggerStop(name); err != nil {
			log.Error("failed to trigger stop: %s", err)
		}
		if err := p.unitManager.Unload(name); err != nil {
			log.Error("failed to unload", err)
		}
		p.unitManager.Reload()
		return nil
	}

	// MetaData as injected by CreatePod.
	om := metav1.ObjectMeta{
		Name:        Pod(name),
		Namespace:   (uf.Contents[kubernetesSection]["Namespace"])[0],
		ClusterName: (uf.Contents[kubernetesSection]["ClusterName"])[0],
		UID:         types.UID((uf.Contents[kubernetesSection]["Id"])[0]),
	}

	containers, initContainers := p.toContainers(stats)
	containerStatuses, initContainerStatuses := p.toContainerStatuses(stats)
	starttime := metav1.NewTime(propertyTimestampToTime(p.unitManager.ServiceProperty(name, "ExecMainStartTimestamp")))

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: om,
		Spec: corev1.PodSpec{
			NodeName:       p.config.NodeName,
			Volumes:        []corev1.Volume{},
			Containers:     containers,
			InitContainers: initContainers,
		},
		Status: corev1.PodStatus{
			HostIP: p.config.NodeInternalIP.String(),
			PodIP:  p.config.NodeInternalIP.String(), // TODO(pires) this won't always be the case but works for now.
			Phase:  toPhase(containerStatuses),       // might need to have pending if pulling done packages... etc?
			Conditions: []corev1.PodCondition{
				{
					Type:               corev1.PodReady,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: starttime,
				},
				{
					Type:               corev1.PodInitialized,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: starttime,
				},
				{
					Type:               corev1.PodScheduled,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: starttime,
				},
			},
			ContainerStatuses:     containerStatuses,
			InitContainerStatuses: initContainerStatuses,
			Message:               string(toPhase(containerStatuses)),
			StartTime:             &starttime,
		},
	}
	return pod
}

func (p *p) toContainers(stats map[string]*unit.State) ([]corev1.Container, []corev1.Container) {
	keys := unitNames(stats)
	var initContainers, containers []v1.Container
	for _, k := range keys {
		s := stats[k]
		u, _ := unit.NewFile(s.UnitData)
		container := v1.Container{
			Name:      Image(k),
			Image:     Image(k), // We not saving the image anywhere, this assume container.Name == container.Image
			Command:   u.Contents["Service"]["ExecStart"],
			Resources: v1.ResourceRequirements{
				/*
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%g", float64(c.CPU))),
						v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dM", containerMemoryMB)),
					},

					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%g", float64(c.CPU*1024/100))),
						v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dM", containerMemoryMB)),
					},
				*/
			},
		}
		if u.Contents[kubernetesSection]["InitContainer"] != nil {
			initContainers = append(initContainers, container)
			continue
		}
		containers = append(containers, container)
	}
	return containers, initContainers
}

func (p *p) toContainerStatuses(stats map[string]*unit.State) ([]corev1.ContainerStatus, []corev1.ContainerStatus) {
	keys := unitNames(stats)
	var initStatuses, statuses []v1.ContainerStatus
	for _, k := range keys {
		s := stats[k]
		u, _ := unit.NewFile(s.UnitData)
		restarts, _ := strconv.Atoi(p.unitManager.ServiceProperty(k, "NRestarts"))
		status := v1.ContainerStatus{
			Name:                 Name(k),
			State:                p.containerState(s),
			LastTerminationState: p.containerState(s),
			Ready:                true, // readiness probes on the container level??
			RestartCount:         int32(restarts),
			Image:                Image(k),
			ImageID:              hash(Image(k)),
			ContainerID:          "pid://" + p.unitManager.ServiceProperty(k, "MainPID"),
		}
		if u.Contents[kubernetesSection]["InitContainer"] != nil {
			initStatuses = append(initStatuses, status)
			continue
		}
		statuses = append(statuses, status)
	}
	return statuses, initStatuses
}

func (p *p) containerState(u *unit.State) v1.ContainerState {
	// systemctl --state=help
	// Look at u.ActiveState at all?
	switch {
	case strings.HasPrefix(u.SubState, "stop"):
		fallthrough
	case u.SubState == "failed" || u.SubState == "exited":
		exitcode := int32(propertyNumberToInt(p.unitManager.ServiceProperty(u.Name, "ExecMainStatus")))
		reason := string(corev1.PodFailed)
		if exitcode == 0 {
			reason = string(corev1.PodSucceeded)
		}
		return v1.ContainerState{
			Terminated: &v1.ContainerStateTerminated{
				ExitCode:    exitcode,
				Reason:      reason,
				Message:     reason,
				StartedAt:   metav1.NewTime(propertyTimestampToTime(p.unitManager.ServiceProperty(u.Name, "ExecMainStartTimestamp"))),
				FinishedAt:  metav1.NewTime(propertyTimestampToTime(p.unitManager.ServiceProperty(u.Name, "ExecMainExitTimestamp"))),
				ContainerID: "pid://" + p.unitManager.ServiceProperty(u.Name, "MainPID"),
			},
		}
	case u.SubState == "dead": // either ran, or waiting to be run
		exitStamp := propertyNumberToInt(p.unitManager.ServiceProperty(u.Name, "ExecMainExitTimestamp"))
		if exitStamp > 0 {
			exitcode := int32(propertyNumberToInt(p.unitManager.ServiceProperty(u.Name, "ExecMainStatus")))
			reason := string(corev1.PodFailed)
			if exitcode == 0 {
				reason = string(corev1.PodSucceeded)
			}
			return v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{
					ExitCode:    exitcode,
					Reason:      reason,
					Message:     reason,
					StartedAt:   metav1.NewTime(propertyTimestampToTime(p.unitManager.ServiceProperty(u.Name, "ExecMainStartTimestamp"))),
					FinishedAt:  metav1.NewTime(propertyTimestampToTime(p.unitManager.ServiceProperty(u.Name, "ExecMainExitTimestamp"))),
					ContainerID: "pid://" + p.unitManager.ServiceProperty(u.Name, "MainPID"),
				},
			}
		}
		fallthrough // fall to condition waiting
	case strings.HasPrefix(u.SubState, "start"):
		fallthrough
	case u.SubState == "condition":
		return v1.ContainerState{
			Waiting: &v1.ContainerStateWaiting{
				Reason:  u.SubState,
				Message: u.SubState,
			},
		}
	case u.SubState == "running" || u.SubState == "auto-restart" || u.SubState == "reload":
		return v1.ContainerState{
			Running: &v1.ContainerStateRunning{
				StartedAt: metav1.NewTime(propertyTimestampToTime(p.unitManager.ServiceProperty(u.Name, "ExecMainStartTimestamp"))),
			},
		}

	default:
		log.Warnf("unhandled substate for %q: %s", u.Name, u.SubState)
		return v1.ContainerState{}
	}
}

func toPhase(status []v1.ContainerStatus) corev1.PodPhase {
	// run through the states, if 1 is waiting return pending.
	running := 0
	terminated := 0
	exitcode := int32(0)
	for _, s := range status {
		s1 := s.State
		if s1.Waiting != nil {
			return corev1.PodPending
		}
		if s1.Running != nil {
			running++
		}
		if s1.Terminated != nil {
			terminated++
			exitcode += s1.Terminated.ExitCode
		}
	}

	if running == len(status) { // all running
		return corev1.PodRunning
	}

	if terminated == len(status) {
		if exitcode == 0 {
			return corev1.PodSucceeded

		}
		return corev1.PodFailed
	}

	return corev1.PodUnknown
}

func unitNames(units map[string]*unit.State) []string {
	keys := make([]string, len(units))
	i := 0
	for k := range units {
		keys[i] = k
		i++
	}
	return sort.StringSlice(keys)
}

const synthUnit = `[Unit]
Description=systemk
Documentation=man:systemk(8)

[Service]
ExecStart=

[Install]
WantedBy=multi-user.target
`

func (p *p) unitfileFromPackageOrSynthesized(c corev1.Container) (*unit.File, error) {
	u, err := p.pkgManager.Unitfile(c.Image)
	if err != nil {
		log.Warnf("failed to find unit file, synthesizing one")
		uf, err := unit.NewFile(synthUnit)
		return uf, err
	}

	log.Debugf("unit file found at %q", u)
	buf, err := ioutil.ReadFile(u)
	if err != nil {
		return nil, err
	}
	uf, err := unit.NewFile(string(buf))
	if err != nil {
		return nil, err
	}
	return uf, nil
}

const kubernetesSection = "X-Kubernetes"

func hash(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func propertyNumberToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func propertyTimestampToTime(s string) time.Time {
	i, _ := strconv.ParseInt(s, 10, 64)
	// i is microseconds as per systemd D-Bus documentation:
	// "Note that properties exposing time values are usually encoded in
	// microseconds (usec) on the bus, even if their corresponding settings
	// in the unit files are in seconds."
	return time.Unix(i/1000000, i%1000000)
}
