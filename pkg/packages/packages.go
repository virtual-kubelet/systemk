package packages

// TODO(pires) shouldn't this be covered by context so it becomes cancelable?

// PackageManager is the interface managing host packages
type PackageManager interface {
	// Install install the given package at the given version, the returned boolean is true.
	// Does nothing if package is already installed, in this case the returned boolean is false.
	Install(pkg, version string) (bool, error)
	// Unitfile returns the location of the unitfile for the given package
	// Returns an error if no unitfiles were found
	Unitfile(pkg string) (string, error)
}
