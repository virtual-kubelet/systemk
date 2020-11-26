package systemd

import (
	"fmt"

	"github.com/miekg/vks/pkg/system"
)

// defaultEnvironment returns the environment that the kubelet uses.
// It returns a list of strings VAR=VALUE.
func (p *P) defaultEnvironment() []string {
	env := []string{}
	env = append(env, fmt.Sprintf("HOSTNAME=%s", system.Hostname()))
	env = append(env, fmt.Sprintf("KUBERNETES_PORT=%d", 443))         // get from provider?
	env = append(env, fmt.Sprintf("KUBERNETES_HOST=%s", "127.0.0.1")) // get from provider?
	return env
}
