package ospkg

import (
	"testing"

	"github.com/virtual-kubelet/systemk/internal/system"
)

func TestDebian(t *testing.T) {
	distro := system.ID()
	switch distro {
	case "debian", "ubuntu":
	default:
		return
	}
	d := new(DebianManager)
	unit, err := d.Unitfile("openssh-server")
	if err != nil {
		// not installed
		return
	}
	if unit == "" {
		// not installed, just bail
		return
	}
	if unit != "/lib/systemd/system/ssh.service" {
		t.Errorf("expected unit to be %s, got %s", "/lib/systemd/system/ssh.service", unit)
	}
}
