package provider

import (
	"testing"

	"github.com/virtual-kubelet/systemk/internal/kubernetes"
)

func TestUidGidFromSecurityContext(t *testing.T) {
	yamlFile := "../testdata/uptimed-config.yaml"
	pod, err := kubernetes.PodFromFile(yamlFile)
	if err != nil {
		t.Error(err)
		return
	}

	uid, gid, err := uidGidFromSecurityContext(pod, 0)
	if err != nil {
		t.Fatal(err)
	}
	if uid != "0" {
		t.Errorf("expected uid to be %q, got %q", "0", uid)
	}
	if gid != "0" {
		t.Errorf("expected uid to be %q, got %q", "0", gid)
	}

	// now map the uid
	uid, gid, err = uidGidFromSecurityContext(pod, 1)
	if err != nil {
		t.Fatal(err)
	}
	if uid != "1" {
		t.Errorf("expected uid to be %q, got %q", "1", uid)
	}
	if gid != "1" {
		t.Errorf("expected uid to be %q, got %q", "1", gid)
	}
}
