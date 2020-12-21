package systemd

import (
	"fmt"
	"net"
	"net/url"

	"github.com/virtual-kubelet/systemk/pkg/system"
)

// defaultEnvironment returns the environment that the kubelet uses.
// It returns a list of strings VAR=VALUE.
func (p *P) defaultEnvironment() []string {
	env := []string{}

	host := "127.0.0.1"
	port := "6444"
	if p.Host != "" {
		url, _ := url.Parse(p.Host)
		host, port, _ = net.SplitHostPort(url.Host)
	}

	env = append(env, fmt.Sprintf("HOSTNAME=%s", system.Hostname()))
	env = append(env, fmt.Sprintf("KUBERNETES_SERVICE_PORT=%s", port))
	env = append(env, fmt.Sprintf("KUBERNETES_SERVICE_HOST=%s", host))

	// These are systemk spefific environment variables. TODO(miek): should this be done at all?
	// SYSTEMK_NODE_INTERNAL_IP: internal address of the node
	// SYSTEMK_NODE_EXTERNAL_IP: external address of the node
	env = append(env, mkEnvVar("NODE_INTERNAL_IP", p.NodeInternalIP.Address))
	env = append(env, mkEnvVar("NODE_EXTERNAL_IP", p.NodeExternalIP.Address))

	return env
}

func mkEnvVar(name, value string) string {
	const s = "SYSTEMK_"
	return s + name + "=" + value
}
