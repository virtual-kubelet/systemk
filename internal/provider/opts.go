// Copyright © 2017 The virtual-kubelet authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Edited by the systemk authors in 2021.

package provider

import (
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/virtual-kubelet/systemk/internal/system"
)

// Defaults for the provider.
var (
	DefaultOperatingSystem       = "Linux" // systemd is only supported on Linux
	DefaultInformerResyncPeriod  = 1 * time.Minute
	DefaultMetricsAddr           = ":10255"
	DefaultListenAddr            = ":10250"
	DefaultPodSyncWorkers        = 10
	DefaultKubeClusterDomain     = "cluster.local"
	DefaultTaintKey              = "virtual-kubelet.io/provider"
	DefaultTaintValue            = "systemk"
	DefaultStreamIdleTimeout     = 30 * time.Second
	DefaultStreamCreationTimeout = 30 * time.Second
	DefaultAllowedPaths          = []string{"/var"}
)

// Opts stores all the configuration options.
// It is used for setting flag values.
//
// You can set the default options by creating a new `Opts` struct and passing
// it into `SetDefaultOpts`
type Opts struct {
	// ServerCertPath is the path to the certificate to secure the kubelet API.
	ServerCertPath string

	// ServerKeyPath is the path to the private key to sign the kubelet API.
	ServerKeyPath string

	// NodeInternalIP is the desired Node internal IP.
	NodeInternalIP net.IP

	// NodeExternalIP is the desired Node external IP.
	NodeExternalIP net.IP

	// AllowedHostPaths is a list of host paths that are allowed to be mounted.
	AllowedHostPaths []string

	// KubeConfigPath is the path to the Kubernetes client configuration.
	KubeConfigPath string

	// KubeClusterDomain is the suffix to append to search domains for the Pods.
	KubeClusterDomain string

	// KubernetesURL is the value to set for the KUBERNETES_SERVICE_* Pod env vars.
	KubernetesURL string

	// ListenAddress is the address to bind for serving requests from the Kubernetes API server.
	ListenAddress string

	// NodeName identifies the Node in the cluster.
	NodeName string

	// DisableTaint disables systemk default taint.
	DisableTaint bool

	// MetricsAddr is the address to bind for serving metrics.
	MetricsAddr string

	// PodSyncWorkers is the number of workers that handle Pod events.
	PodSyncWorkers int

	// InformerResyncPeriod is the interval between relisting of Kubernetes resources.
	// This is important as it serves as a recovery mechanism in case systemk lost any
	// events related to the resources it's watching, eg due to network partition.
	InformerResyncPeriod time.Duration

	// StartupTimeout is how long to wait for systemk to start.
	StartupTimeout time.Duration

	// StreamIdleTimeout is the maximum time a streaming connection
	// can be idle before the connection is automatically closed.
	StreamIdleTimeout time.Duration

	// StreamCreationTimeout is the maximum time for streaming connection.
	StreamCreationTimeout time.Duration

	// Version carries the systemk version.
	Version string
}

// SetDefaultOpts sets default options for unset values of the passed in option struct.
// Fields that are already set will not be modified.
func SetDefaultOpts(opts *Opts) {
	if len(opts.AllowedHostPaths) == 0 {
		opts.AllowedHostPaths = DefaultAllowedPaths
	}

	if opts.NodeName == "" {
		// If flag isn't set, try environment but fallback to systemd.
		opts.NodeName = getEnvOrDefault("HOSTNAME", system.Hostname())
	}

	if opts.InformerResyncPeriod == 0 {
		opts.InformerResyncPeriod = DefaultInformerResyncPeriod
	}

	if opts.PodSyncWorkers <= 0 {
		opts.PodSyncWorkers = DefaultPodSyncWorkers
	}

	if _, err := net.ResolveTCPAddr("tcp", opts.ListenAddress); err != nil {
		opts.ListenAddress = DefaultListenAddr
	}

	if _, err := net.ResolveTCPAddr("tcp", opts.MetricsAddr); err != nil {
		opts.MetricsAddr = DefaultMetricsAddr
	}

	// TODO(pires) consider using miekg/dns isDomainName instead of this poor validation.
	if opts.KubeClusterDomain == "" {
		opts.KubeClusterDomain = DefaultKubeClusterDomain
	}

	if opts.KubeConfigPath == "" {
		// Prefer environment variable.
		opts.KubeConfigPath = os.Getenv("KUBECONFIG")
		// Otherwise, try homedir.
		if opts.KubeConfigPath == "" {
			// Given this code is only supportedon Linux, $HOME should be set.
			opts.KubeConfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}
	}

	if opts.StreamIdleTimeout == 0 {
		opts.StreamIdleTimeout = DefaultStreamIdleTimeout
	}

	if opts.StreamCreationTimeout == 0 {
		opts.StreamCreationTimeout = DefaultStreamCreationTimeout
	}

	if len(opts.AllowedHostPaths) == 0 {
		opts.AllowedHostPaths = DefaultAllowedPaths
	}
}

// getEnvOrDefault returns environment variable value or if unset, a default value.
func getEnvOrDefault(key, defaultValue string) string {
	value, found := os.LookupEnv(key)
	if found {
		return value
	}

	return defaultValue
}
