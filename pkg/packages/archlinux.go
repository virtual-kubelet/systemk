package packages

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/miekg/vks/pkg/unit"
)

// ArchlinuxPackageManager implemtents the PackageManager interface for an Archlinux system
type ArchlinuxPackageManager struct{}

const (
	pacmanCommand                       = "/usr/bin/pacman"
	archlinuxSystemdUnitfilesPathPrefix = "/usr/lib/systemd/system/"
)

// Install install the given package at the given version
// Does nothing if package is already installed
func (p *ArchlinuxPackageManager) Install(pkg, version string) error {
	checkCmdArgs := []string{"-Qi", pkg}
	checkCmd := exec.Command(pacmanCommand, checkCmdArgs...)

	err := checkCmd.Run()
	if err == nil {
		// package exists
		// TODO: check installed version
		return nil
	}

	// no way to specify a package version in arch
	installCmdArgs := []string{"-S", "--noconfirm", pkg}
	installCmd := exec.Command(aptGetCommand, installCmdArgs...)

	_, err = installCmd.CombinedOutput()
	return err
}

// Unitfile returns the location of the unitfile for the given package
// Returns an error if no unitfiles were found
func (p *ArchlinuxPackageManager) Unitfile(pkg string) (string, error) {
	cmd := exec.Command(pacmanCommand, "-Ql", pkg)
	buf, err := cmd.Output()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewReader(buf))
	for scanner.Scan() {
		splitLine := strings.Split(scanner.Text(), " ")
		if len(splitLine) != 2 {
			return "", fmt.Errorf("error checking package files")
		}
		if !strings.HasPrefix(splitLine[1], archlinuxSystemdUnitfilesPathPrefix) {
			continue
		}
		if strings.HasSuffix(splitLine[1], unit.ServiceSuffix) {
			return splitLine[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	// if not found, scan the directory to see if we can spot one
	basicPath := archlinuxSystemdUnitfilesPathPrefix + pkg + unit.ServiceSuffix
	if _, err := os.Stat(basicPath); err != nil {
		return "", err
	}
	return basicPath, nil
}
