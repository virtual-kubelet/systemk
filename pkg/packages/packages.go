package packages

// PackageManager is the interface managing host packages
type PackageManager interface {
	// Install install the given package at the given version
	// Does nothing if package is already installed
	Install(pkg, version string) error
	// Unitfile returns the location of the unitfile for the given package
	// Returns an error if no unitfiles were found
	Unitfile(pkg string) (string, error)
}
