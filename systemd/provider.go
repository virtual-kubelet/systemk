package systemd

import (
	"fmt"

	"github.com/miekg/vks/pkg/manager"
	"github.com/miekg/vks/pkg/packages"
	"github.com/miekg/vks/pkg/system"
	"github.com/virtual-kubelet/node-cli/provider"
)

// P is a systemd provider.
type P struct {
	m   *manager.UnitManager
	pkg packages.PackageManager
}

// New returns a new systemd provider.
func New() (*P, error) {
	m, err := manager.New("/tmp/bla", false)
	if err != nil {
		return nil, err
	}
	p := &P{m: m}
	switch system.ID() {
	default:
		return nil, fmt.Errorf("unsupported system")
	case "debian", "ubuntu":
		p.pkg = new(packages.DebianPackageManager)
	case "arch":
		p.pkg = new(packages.ArchlinuxPackageManager)
	}
	return p, nil
}

var _ provider.Provider = new(P)
