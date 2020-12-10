package systemd

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/virtual-kubelet/systemk/pkg/unit"
	corev1 "k8s.io/api/core/v1"
)

// commandAndArgs returns an updated ExecStart strings slice taking the pod's Command and Args
// into account.
func commandAndArgs(uf *unit.File, c corev1.Container) []string {
	// If command and/or args are given we need to override the ExecStart
	// Bit execStart should be a string slice, but isn't returned like this, so this involves some string wrangling
	// to get things right.
	execStart := uf.Contents["Service"]["ExecStart"] // check if exists...?
	cmdargs := execStart
	if len(execStart) == 1 {
		cmdargs = strings.Fields(execStart[0])
	}

	if c.Command != nil {
		if cmd := c.Command[0]; !path.IsAbs(cmd) {
			fullpath, err := exec.LookPath(cmd)
			if err == nil {
				c.Command[0] = fullpath
			}
			// if this errored the unit will fail, so fail there instead of erroring here.
		}
		cmdargs[0] = strings.Join(c.Command, " ") // some args might be included here/. Does this needs quoting?
	}
	if c.Args != nil {
		cmdargs = cmdargs[:1]
		// if they contain space the need to be quoted. Maybe always quote?
		for _, a := range c.Args {
			cmdargs = append(cmdargs, fmt.Sprintf("%q", a))

		}
	}

	return cmdargs
}
