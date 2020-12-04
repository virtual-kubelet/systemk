package packages

import (
	"fmt"
)

// TestPackageManager is a noop manager.
type TestPackageManager struct{}

func (t *TestPackageManager) Install(pkg, version string) (error, bool) {
	return nil, false
}

func (t *TestPackageManager) Unitfile(pkg string) (string, error) {
	return "", fmt.Errorf("not found: %s", pkg)
}

func (t *TestPackageManager) Clean(pkg string) string { return pkg }
