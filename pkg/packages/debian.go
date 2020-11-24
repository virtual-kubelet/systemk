package packages

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/miekg/vks/pkg/unit"
)

// DebianPackageManager implemtents the PackageManager interface for a Debian system
type DebianPackageManager struct{}

const (
	aptGetCommand                    = "/usr/bin/apt-get"
	aptCacheCommand                  = "/usr/bin/apt-cache"
	dpkgCommand                      = "/usr/bin/dpkg"
	debianSystemdUnitfilesPathPrefix = "/lib/systemd/system/"
)

// Install install the given package at the given version
// Does nothing if package is already installed
func (p *DebianPackageManager) Install(pkg, version string) (error, bool) {
	checkCmdArgs := []string{
		"show",
		pkg,
	}
	checkCmd := exec.Command(aptCacheCommand, checkCmdArgs...)

	err := checkCmd.Run()
	if err == nil {
		// package exists
		// TODO: check installed version
		return nil, false
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() != 100 { // 100 is No packages found
			return fmt.Errorf("failed to check existence of package %s: %w", pkg, err), false
		}
	}

	policyfile, err := policy()
	if err != nil {
		return err, false
	}
	defer os.Remove(policyfile)

	pkgToInstall := pkg
	if version != "" {
		pkgToInstall = fmt.Sprintf("%s=%s*", pkg, version)
	}
	installCmdArgs := []string{"-qq", "--force-yes", "install", pkgToInstall}
	installCmd := exec.Command(aptGetCommand, installCmdArgs...)
	installCmd.Env = append(installCmd.Env, fmt.Sprintf("POLICYRCD=%s", policyfile))

	out, err := installCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install: %s\n%s", err, out), false
	}
	return nil, true
}

// policy writes a small script to disk, that only does exit 0
func policy() (string, error) {
	f, err := ioutil.TempFile("", "policy-donotstart")
	if err != nil {
		return "", err
	}
	f.WriteString("#!/bin/sh\n")
	f.WriteString("exit 101\n")
	if err := os.Chmod(f.Name(), 0755); err != nil {
		return "", err
	}
	return f.Name(), nil
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
		if !strings.HasPrefix(scanner.Text(), debianSystemdUnitfilesPathPrefix) {
			continue
		}
		if strings.HasSuffix(scanner.Text(), unit.ServiceSuffix) {
			return scanner.Text(), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	// if not found, scan the directory to see if we can spot one
	basicPath := debianSystemdUnitfilesPathPrefix + pkg + unit.ServiceSuffix
	if _, err := os.Stat(basicPath); err != nil {
		return "", err
	}
	return basicPath, nil
}
