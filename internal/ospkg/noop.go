package ospkg

import (
	"fmt"
)

// NoopManager implements a no-op of the Manager interface.
// Its purpose is to enable scenarios where no package handling is required,
// i.e. the necessary executables are already available on the host.
type NoopManager struct{}

var _ Manager = (*NoopManager)(nil)

func (p *NoopManager) Setup() error                              { return nil }
func (p *NoopManager) Install(pkg, version string) (bool, error) { return true, nil }
func (p *NoopManager) Unitfile(pkg string) (string, error) {
	// This is fine as pod creation will synthesize a unit file.
	return "", fmt.Errorf("noop")
}
