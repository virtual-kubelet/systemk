package packages

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/miekg/vks/pkg/unit"
)

// DebianPackageManager implemtents the PackageManager interface for a Debian system
type DebianPackageManager struct {
	sync.RWMutex
}

const (
	aptGetCommand                    = "/usr/bin/apt-get"
	dpkgCommand                      = "/usr/bin/dpkg"
	debianSystemdUnitfilesPathPrefix = "/lib/systemd/system/"
)

// Install install the given package at the given version.
func (p *DebianPackageManager) Install(pkg, version string) error {
	p.Lock()
	defer p.Unlock()

	log.Printf("Debian: Install: %s=%s", pkg, version)
	setup()
	pkgToInstall := pkg
	if version != "" {
		pkgToInstall = fmt.Sprintf("%s=%s*", pkg, version)
	}
	installCmdArgs := []string{"-qq", "--assume-yes", "install", pkgToInstall}
	installCmd := exec.Command(aptGetCommand, installCmdArgs...)
	installCmd.Env = append(installCmd.Env, "POLICYRCD=/tmp/policy-donotstart")

	println("Running apt")
	out, err := installCmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to install: %s\n%s", err, out)
	} else {
		log.Printf("Installed %s", pkgToInstall)
	}
	return err
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

func setup() error {
	err := ioutil.WriteFile("/tmp/policy-donotstart", []byte("exit 101\n"), 0755)
	if err != nil {
		log.Printf("Failed setup: %s", err)
	}
	return nil
}
