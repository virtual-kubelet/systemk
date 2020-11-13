package provider

import (
	"context"
	"fmt"

	"github.com/miekg/vks/pkg/systemd"
)

type P struct {
	um *systemd.UnitManager
}

func New() (*P, error) {
	um, err := systemd.New("/tmp/bla", false)
	if err != nil {
		return nil, err
	}
	return &P{um}, nil
}

//func (p *P) GetPods(context.Context) ([]*corev1.Pod, error) {}

func (p *P) GetPods(context.Context) error {
	unitstats, err := p.um.ListUnits()
	if err != nil {
		return err
	}

	for i := range unitstats {
		fmt.Printf("%+v\n", unitstats[i])
	}
	return nil
}
