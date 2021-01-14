package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/virtual-kubelet/systemk/pkg/unit"
	vklog "github.com/virtual-kubelet/virtual-kubelet/log"
)

// DebianPackageManager implemtents the PackageManager interface for a Debian system
type DebianPackageManager struct{}

const (
	aptGetCommand                    = "/usr/bin/apt-get"
	dpkgCommand                      = "/usr/bin/dpkg"
	debianSystemdUnitfilesPathPrefix = "/lib/systemd/system/"
)

// Install install the given package at the given version
// Does nothing if package is already installed or if we're not handling
// an actual OS package, e.g. /bin/bash
func (p *DebianPackageManager) Install(pkg, version string) (bool, error) {
	vklog.G(context.TODO()).Infof("Checking if %q is installed", Clean(pkg))
	if path.IsAbs(pkg) {
		return false, nil
	}
	checkCmd := exec.Command(dpkgCommand, "-s", Clean(pkg))

	if err := checkCmd.Run(); err == nil {
		return false, nil
	}

	installCmd := new(exec.Cmd)
	switch {
	case strings.HasPrefix(pkg, "https://"):
		pkgToInstall, err := fetch(pkg[6:], "")
		if err != nil {
			return false, err
		}
		installCmdArgs := []string{"-i", pkgToInstall}
		installCmd = exec.Command(dpkgCommand, installCmdArgs...)
	default:
		pkgToInstall := pkg
		if version != "" {
			pkgToInstall = fmt.Sprintf("%s=%s*", pkg, version)
		}
		installCmdArgs := []string{"-qq", "--assume-yes", "--no-install-recommends", "install", pkgToInstall}
		installCmd = exec.Command(aptGetCommand, installCmdArgs...)
	}

	policyfile, err := policy()
	if err != nil {
		return false, err
	}
	defer os.Remove(policyfile)

	installCmd.Env = []string{fmt.Sprintf("POLICYRCD=%s", policyfile)} // this effectively clears the env for this command, add stuff back in
	for _, env := range []string{"PATH", "HOME", "LOGNAME"} {
		installCmd.Env = append(installCmd.Env, env+"="+os.Getenv(env))
	}

	vklog.G(context.TODO()).Infof("Running %s", installCmd)
	if out, err := installCmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("failed to install: %s\n%s", err, out)
	}
	return true, nil
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
	f.Close() // prevent "text file busy"
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
