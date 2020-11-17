package systemd

import "testing"

func TestNameSplitting(t *testing.T) {
	name := "vks.default.openssh-server.openssh-server-image.d1d316c0-d10d-4653-940a-31d808efb4a7.service"
	if x := Image(name); x != "openssh-server-image" {
		t.Errorf("expected Image to be %s, got %s", "openssh-server-image", x)
	}
	if x := Pod(name); x != "openssh-server" {
		t.Errorf("expected Pod to be %s, got %s", "openssh-server", x)
	}
	if x := Name(name); x != "default.openssh-server" {
		t.Errorf("expected Name to be %s, got %s", "default.openssh-server", x)
	}
	if x := Namespace(name); x != "default" {
		t.Errorf("expected Namespace to be %s, got %s", "default", x)
	}
	if x := UID(name); x != "d1d316c0-d10d-4653-940a-31d808efb4a7" {
		t.Errorf("expected UID to be %s, got %s", "d1d316c0-d10d-4653-940a-31d808efb4a7", x)
	}
}
