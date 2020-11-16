package systemd

import "testing"

func TestUidFromUnitName(t *testing.T) {
	name := ImageNameToUnitNameUID("default", "openssh-server", "we4rw4")
	uid := UidFromUnitName(name)
	if uid != "we4rw4" {
		t.Errorf("expected %s, got %s", "we4rw4", uid)
	}
}
