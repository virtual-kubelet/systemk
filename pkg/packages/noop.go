package packages

import (
	"fmt"
)

// NoopPackageManager implements a no-op of the PackageManager interface.
// Its purpose is to enable scenarios where no package handling is required,
// i.e. the necessary executables are already available on the host.
type NoopPackageManager struct{}

func (p *NoopPackageManager) Setup() error                              { return nil }
func (p *NoopPackageManager) Install(pkg, version string) (bool, error) { return true, nil }
func (p *NoopPackageManager) Clean(pkg string) string                   { return pkg }
func (p *NoopPackageManager) Unitfile(pkg string) (string, error) {
	// This is fine as pod creation will synthesize a unit file.
	return "", fmt.Errorf("noop")
}
