package provider

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/virtual-kubelet/systemk/internal/unit"
	corev1 "k8s.io/api/core/v1"
)

// commandAndArgs returns an updated ExecStart strings slice taking the pod's Command and Args
// into account. Any errors will lead to an invalid unit, but the control plane deals with those.
func commandAndArgs(uf *unit.File, c corev1.Container) []string {
	// If command and/or args are given we need to override the ExecStart
	execStart := uf.Contents["Service"]["ExecStart"]
	cmdargs := execStart
	if len(execStart) == 1 {
		cmdargs = strings.Fields(execStart[0])
	}
	if len(cmdargs) == 0 {
		cmdargs = make([]string, 1)
	}

	if c.Command != nil {
		if cmd := c.Command[0]; !path.IsAbs(cmd) {
			fullpath, err := exec.LookPath(cmd)
			if err == nil {
				c.Command[0] = fullpath
			}
		}
		cmdargs[0] = strings.Join(c.Command, " ") // TODO(miek) some args might be included here. Does this needs quoting?
	}
	if c.Args != nil {
		if len(cmdargs) > 0 {
			cmdargs = cmdargs[:1]
		}
		// quote the arguments
		for _, a := range c.Args {
			cmdargs = append(cmdargs, fmt.Sprintf("%q", a))

		}
	}

	return cmdargs
}
