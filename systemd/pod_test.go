package systemd

import "testing"

func TestUidFromUnitName(t *testing.T) {
	name := PodToUnitName("default", "openssh-server", "openssh-server", "we4rw4")
	uid := UidFromUnitName(name)
	if uid != "we4rw4" {
		t.Errorf("expected %s, got %s", "we4rw4", uid)
	}
}
