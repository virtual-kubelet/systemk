package ospkg

import (
	vklogv2 "github.com/virtual-kubelet/virtual-kubelet/log/klogv2"
)

// log is the global logger for OS package management.
var log = vklogv2.New(nil)

// Manager represents OS package management.
type Manager interface {
	// Install install the given package at the given version, the returned boolean is true.
	// Does nothing if package is already installed, in this case the returned boolean is false.
	Install(pkg, version string) (bool, error)
	// Unitfile returns the location of the unitfile for the given package
	// Returns an error if no unitfiles were found
	Unitfile(pkg string) (string, error)
}
