package systemd

import (
	"github.com/miekg/vks/pkg/manager"
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
