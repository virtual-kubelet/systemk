package systemd

import (
	"github.com/miekg/vks/pkg/manager"
	"github.com/virtual-kubelet/node-cli/provider"
)

type P struct {
	m *manager.UnitManager
}

func NewProvider() (*P, error) {
	m, err := manager.New("/tmp/bla", false)
	if err != nil {
		return nil, err
	}
	return &P{m}, nil
}

var _ provider.Provider = new(P)
