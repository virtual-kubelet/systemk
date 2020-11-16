package vks

// PackageManager abstract away the differences between Linux/Unix distributions.
type PackageManager interface {
	Install(name, version string) error   // Install installs a package with name, possibly with version (if given).
	UnitFile(name string) (string, error) // UnitFile returns the path of the unit file we're interested in.
}
