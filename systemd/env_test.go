package systemd

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestProviderIPEnvironment(t *testing.T) {
	p := new(P)
	p.NodeInternalIP = &corev1.NodeAddress{Address: "192.168.1.1", Type: corev1.NodeInternalIP}
	p.NodeExternalIP = &corev1.NodeAddress{Address: "172.16.0.1", Type: corev1.NodeExternalIP}

	env := p.defaultEnvironment()
	found := 0
	for _, e := range env {
		if e == "SYSTEMK_NODE_INTERNAL_IP=192.168.1.1" {
			found++
		}
		if e == "SYSTEMK_NODE_EXTERNAL_IP=172.16.0.1" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("failed to find SYSTEMK_NODE_INTERNAL_IP or SYSTEMK_NODE_EXTERNAL_IP")
	}
}
