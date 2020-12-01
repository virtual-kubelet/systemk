package systemd

import (
	"fmt"
	"log"
	"os"

	"github.com/miekg/vks/pkg/manager"
	"github.com/miekg/vks/pkg/packages"
	"github.com/miekg/vks/pkg/system"
	vkmanager "github.com/virtual-kubelet/node-cli/manager"
	"github.com/virtual-kubelet/node-cli/provider"
	corev1 "k8s.io/api/core/v1"
)

// unitDir is where vks stores the modified unit files.
var unitDir = "/var/run/vks"

// P is a systemd provider.
type P struct {
	m   *manager.UnitManager
	pkg packages.PackageManager
	rm  *vkmanager.ResourceManager

	Addresses     []corev1.NodeAddress
	DaemonPort    int32
	ClusterDomain string
}

// New returns a new systemd provider.
func New(cfg provider.InitConfig) (*P, error) {
	if err := os.MkdirAll(unitDir, 0750); err != nil {
		return nil, err
	}
	m, err := manager.New(unitDir, false)
	if err != nil {
		return nil, err
	}
	p := &P{m: m}
	switch system.ID() {
	default:
		return nil, fmt.Errorf("unsupported system")
	case "debian", "ubuntu":
		p.pkg = new(packages.DebianPackageManager)

		// Just installed pre-requisites instead of pointing to the docs.
		log.Printf("Installing %s, to prevent installed daemons from starting", "policyrcd-script-zg2")
		if err, _ := p.pkg.Install("policyrcd-script-zg2", ""); err != nil {
			log.Printf("Failed to install %s, %s. Continuing anyway", "policyrcd-script-zg2", err)
		}

	case "arch":
		p.pkg = new(packages.ArchlinuxPackageManager)
	}

	p.rm = cfg.ResourceManager
	p.DaemonPort = cfg.DaemonPort
	p.ClusterDomain = cfg.KubeClusterDomain
	return p, nil
}

var _ provider.Provider = new(P)
