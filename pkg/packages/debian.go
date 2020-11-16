package packages

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DebianPackageManager implemtents the PackageManager interface for a Debian system
type DebianPackageManager struct{}

const (
	aptGetCommand              = "/usr/bin/apt-get"
	aptCacheCommand            = "/usr/bin/apt-cache"
	dpkgCommand                = "/usr/bin/dpkg"
	systemdUnitfilesPathPrefix = "/lib/systemd/system/"
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

	installCmdArgs := []string{"-qq", "--force-yes", "install"}
	if version != "" {
		installCmdArgs = append(installCmdArgs, fmt.Sprintf("%s=%s*", pkg, version))
	}
	installCmd := exec.Command(aptGetCommand, installCmdArgs...)

	_, err = installCmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

// Unitfile returns the location of the unitfile for the given package
// Returns an error if no unitfiles were found
func (p *DebianPackageManager) Unitfile(pkg string) (string, error) {
	cmd := exec.Command(dpkgCommand, "-L", pkg)
	buf, err := cmd.Output()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewReader(buf))
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), systemdUnitfilesPathPrefix) {
			continue
		}
		if strings.HasSuffix(scanner.Text(), SystemdUnitfileSuffix) {
			return scanner.Text(), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	// if not found, scan the directory to see if we can spot one
	basicPath := systemdUnitfilesPathPrefix + pkg + SystemdUnitfileSuffix
	if _, err := os.Stat(basicPath); err != nil {
		return "", err
	}
	return basicPath, nil
}
