package ospkg

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/virtual-kubelet/systemk/internal/unit"
)

// ArchLinuxManager manages packages on Arch Linux.
type ArchLinuxManager struct{}

var _ Manager = (*ArchLinuxManager)(nil)

const (
	pacmanCommand                       = "/usr/bin/pacman"
	archlinuxSystemdUnitfilesPathPrefix = "/usr/lib/systemd/system/"
)

func (p *ArchLinuxManager) Setup() error {
	return nil
}

func (p *ArchLinuxManager) Install(pkg, version string) (bool, error) {
	log.WithField("os", "archlinux").Infof("checking if %q is installed", Clean(pkg))
	if path.IsAbs(pkg) {
		return false, nil
	}

	checkCmdArgs := []string{"-Qi", pkg}
	checkCmd := exec.Command(pacmanCommand, checkCmdArgs...)

	err := checkCmd.Run()
	if err == nil {
		// package exists
		// TODO: check installed version
		return false, nil
	}

	// no way to specify a package version in arch
	installCmdArgs := []string{"-S", "--noconfirm", pkg}
	installCmd := exec.Command(aptGetCommand, installCmdArgs...)

	_, err = installCmd.CombinedOutput()
	return true, err
}

func (p *ArchLinuxManager) Unitfile(pkg string) (string, error) {
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
