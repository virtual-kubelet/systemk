package systemd

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/systemk/pkg/system"
	"github.com/virtual-kubelet/systemk/pkg/unit"
)

func TestNewUnit(t *testing.T) {
	distro := system.ID()
	switch distro {
	case "debian", "ubuntu":
	default:
		return
	}

	dir, err := ioutil.TempDir(".", "units")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up
	unitDir = dir
	p, err := New(provider.InitConfig{})
	if err != nil {
		t.Fatal(err)
	}

	u, err := p.pkg.Unitfile("openssh-server")
	if err != nil {
		return // not installed
	}
	buf, err := ioutil.ReadFile(u)
	if err != nil {
		t.Fatal(err)
	}
	uf, err := unit.New(string(buf))
	if err != nil {
		t.Fatal(err)
	}
	const want = "OpenBSD Secure Shell server"
	if x := uf.Description(); x != want {
		t.Fatalf("want description %s, got %s", want, x)
	}

	/* requires root access
	if err := p.m.Load("systemk-openssh-server.service", *uf); err != nil {
		t.Fatal(err)
	}
	*/
}
