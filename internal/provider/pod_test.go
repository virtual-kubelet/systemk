package provider

import "testing"

func TestNameSplitting(t *testing.T) {
	name := "systemk.default.openssh-server.openssh-server-container.service"
	if x := Container(name); x != "openssh-server-container" {
		t.Errorf("expected Image to be %s, got %s", "openssh-server-container", x)
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
}
