package provider

import (
	"fmt"
	"net"
	"net/url"
)

// defaultEnvironment returns a list of strings formatted as VAR=VALUE
// to be appended to a systemd unit's environment variables.
func (p *p) defaultEnvironment() []string {
	var env []string
	host := "127.0.0.1"
	port := "6444"
	if p.kubernetesURL != "" {
		kubernetesURL, _ := url.Parse(p.kubernetesURL)
		host, port, _ = net.SplitHostPort(kubernetesURL.Host)
	}
	env = append(env, fmt.Sprintf("HOSTNAME=%s", p.config.NodeName))
	env = append(env, fmt.Sprintf("KUBERNETES_SERVICE_PORT=%s", port))
	env = append(env, fmt.Sprintf("KUBERNETES_SERVICE_HOST=%s", host))

	// These are systemk specific environment variables.
	//
	// TODO(miek) should this be done at all?
	//
	// NOTE(pires) what if there are multiple IPs, eg dual-stack node (both IPv4 and IPv6)?
	//
	// SYSTEMK_NODE_INTERNAL_IP: internal address of the node
	// SYSTEMK_NODE_EXTERNAL_IP: external address of the node
	env = append(env, mkEnvVar("NODE_INTERNAL_IP", p.config.NodeInternalIP.String()))
	env = append(env, mkEnvVar("NODE_EXTERNAL_IP", p.config.NodeExternalIP.String()))

	return env
}

func mkEnvVar(name, value string) string {
	const s = "SYSTEMK_"
	return s + name + "=" + value
}
