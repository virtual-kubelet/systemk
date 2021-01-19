package provider

import (
	"testing"
)

func TestProviderIPEnvironment(t *testing.T) {
	p := new(p)
	p.config = &Opts{
		NodeInternalIP: []byte{192, 168, 1, 1},
		NodeExternalIP: []byte{172, 16, 0, 1},
	}
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
