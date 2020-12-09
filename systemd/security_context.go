package systemd

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

func UidGidFromSecurityContext(pod *corev1.Pod) (uid string, gid string) {
	if x := pod.Spec.SecurityContext; x != nil {
		if x.RunAsUser != nil {
			uid = strconv.FormatInt(*x.RunAsUser, 10)
		}
		if x.RunAsGroup != nil {
			gid = strconv.FormatInt(*x.RunAsGroup, 10)
		}
	}
	return uid, gid
}
