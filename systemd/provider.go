package systemd

import (
	"context"
	"fmt"

	"github.com/miekg/vks/pkg/manager"
	corev1 "k8s.io/api/core/v1"
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

func (p *P) GetPods(_ context.Context) ([]*corev1.Pod, error) {
	unitstats, err := p.m.ListUnits()
	if err != nil {
		return nil, err
	}

	for i := range unitstats {
		fmt.Printf("%+v\n", unitstats[i])
	}
	return nil, nil
}
