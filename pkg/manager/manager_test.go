package manager

import (
	"testing"
)

func TestListUnits(t *testing.T) {
	um, err := New("/tmp/bla", false) // secure temp dir, also for virtual-kubelet
	if err != nil {
		t.Fatal(err)
	}
	state, err := um.GetUnitStates("vks.")
	if err != nil {
		t.Fatal(err)
	}
	// len state, but we should start one first.
}
