package packages

import (
	"fmt"
	"os"
	"os/exec"
)

// DebianPackageManager implemtents the PackageManager interface for a Debian system
type DebianPackageManager struct{}

const (
	aptGetCommand              = "apt-get"
	aptCacheCommand            = "apt-cache"
	systemdUnitfilesPathPrefix = "/lib/systemd/system/"
	systemdUnitfileSuffix      = ".service"
)

// Install install the given package at the given version
// Does nothing if package is already installed
func (p *DebianPackageManager) Install(pkg, version string) error {
	checkCmdArgs := []string{
		"show",
		pkg,
	}
	checkCmd := exec.Command(aptCacheCommand, checkCmdArgs...)

	err := checkCmd.Run()
	if err == nil {
		// package exists
		// TODO: check installed version
		return nil
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() != 100 { // 100 is No packages found
			return fmt.Errorf("failed to check existence of package %s: %w", pkg, err)
		}
	}

	installCmdArgs := []string{
		"-qq",
		"--force-yes",
		"install",
		fmt.Sprintf("%s=%s*", pkg, version),
	}
	installCmd := exec.Command(aptGetCommand, installCmdArgs...)

	// logging, etc metrics
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

// Unitfile returns the location of the unitfile for the given package
// Returns an error if no unitfiles were found
func (p *DebianPackageManager) Unitfile(pkg string) (string, error) {
	basicPath := systemdUnitfilesPathPrefix + pkg + systemdUnitfileSuffix
	_, err := os.Stat(basicPath)
	if err == nil {
		return basicPath
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to stat on %s: %w", basicPath, err)
	}

	// TODO handle corner cases like openssh
	return "", fmt.Errorf("not implemtented")
}
