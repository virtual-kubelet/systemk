package systemd

import (
	"fmt"

	"github.com/virtual-kubelet/systemk/pkg/system"
)

// defaultEnvironment returns the environment that the kubelet uses.
// It returns a list of strings VAR=VALUE.
func (p *P) defaultEnvironment() []string {
	env := []string{}
	env = append(env, fmt.Sprintf("HOSTNAME=%s", system.Hostname()))
	env = append(env, fmt.Sprintf("KUBERNETES_SERVICE_PORT=%d", 6444))        // get from provider/flag?
	env = append(env, fmt.Sprintf("KUBERNETES_SERVICE_HOST=%s", "127.0.0.1")) // get from provider/flag?
	return env
}
