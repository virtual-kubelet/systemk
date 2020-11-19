package systemd

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func activeStateToPhase(status string) corev1.PodPhase {
	// See https://superuser.com/questions/896812/all-systemd-states
	// https://www.freedesktop.org/software/systemd/man/systemd.html
	switch status {
	case "active":
		return corev1.PodRunning
	case "inactive":
		return corev1.PodFailed
	case "failed":
		return corev1.PodFailed
	case "activating":
		return corev1.PodPending
	case "deactivating":
		return corev1.PodPending
	}

	return corev1.PodUnknown
}

func activeStateToPodConditions(status string, transitiontime metav1.Time) []corev1.PodCondition {
	switch status {
	case "active":
		return []corev1.PodCondition{
			{
				Type:               corev1.PodReady,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: transitiontime,
			},
			{
				Type:               corev1.PodInitialized,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: transitiontime,
			},
			{
				Type:               corev1.PodScheduled,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: transitiontime,
			},
		}
	}
	return []corev1.PodCondition{}
}
