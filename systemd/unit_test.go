package systemd

import (
	"io/ioutil"
	"testing"

	"github.com/miekg/vks/pkg/system"
	"github.com/miekg/vks/pkg/unit"
)

func TestNewUnit(t *testing.T) {
	distro := system.ID()
	switch distro {
	case "debian", "ubuntu":
	default:
		return
	}
	p, err := NewProvider()
	if err != nil {
		t.Fatal(err)
	}

	u, err := p.Pkg.Unitfile("openssh-server")
	if err != nil {
		t.Fatal(err)
	}
	buf, err := ioutil.ReadFile(u)
	if err != nil {
		t.Fatal(err)
	}
	uf, err := unit.NewUnitFile(string(buf))
	if err != nil {
		t.Fatal(err)
	}
	const want = "OpenBSD Secure Shell server"
	if x := uf.Description(); x != want {
		t.Fatalf("want description %s, got %s", want, x)
	}

	/* requires root access
	if err := p.m.Load("vks-openssh-server.service", *uf); err != nil {
		t.Fatal(err)
	}
	*/
}
