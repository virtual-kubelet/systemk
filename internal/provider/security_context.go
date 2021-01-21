package provider

import (
	"os/user"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

// uidGidFromSecurityContext returns the uid and gid (as a string) from the SecurityContext.
// If windowsOptions are set, the uid and gid *names* found there are returned. This takes
// precedence over the values runAsUser and runAsGroup.
// If the uid is found, but gid is not, the primary group for uid is searched and returned.
func uidGidFromSecurityContext(pod *corev1.Pod) (uid string, gid string) {
	if pod.Spec.SecurityContext == nil {
		return "", ""
	}
	u := &user.User{}
	s := pod.Spec.SecurityContext
	if s.RunAsUser != nil {
		uid = strconv.FormatInt(*s.RunAsUser, 10)
		u, _ = user.LookupId(uid)
	}
	if s.RunAsGroup != nil {
		gid = strconv.FormatInt(*s.RunAsGroup, 10)
	}
	if s.WindowsOptions != nil {
		if s.WindowsOptions.RunAsUserName != nil {
			uid = *s.WindowsOptions.RunAsUserName
			u, _ = user.Lookup(uid)
			if u != nil {
				uid = u.Uid
			}
		}
	}

	// if uid is set, but gid is return the default group for uid.
	if uid != "" && gid == "" {
		if u != nil {
			gid = u.Gid
		}
	}

	return uid, gid
}
