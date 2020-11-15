package systemd

import (
	"os/exec"
)

func install(name string) error {
	cmd := exec.Command("apt-get", "-qq", "--force-yes", "install", name)
	// logging, etc metrics
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}
