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

package cmd

import (
	"flag"
	"net"
	"os"

	"github.com/spf13/pflag"
	"github.com/virtual-kubelet/systemk/internal/provider"
	vklogv2 "github.com/virtual-kubelet/virtual-kubelet/log/klogv2"
	"k8s.io/klog/v2"
)

// A global logger for this package is declared here since this is the place
// where we tie systemk logging to klogv2 (see the imports).
var log = vklogv2.New(nil)

func installFlags(flags *pflag.FlagSet, c *provider.Opts) {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "systemk"
	}

	flags.StringVar(&c.KubeConfigPath, "kubeconfig", "", "cluster client configuration")
	flags.StringVar(&c.KubeClusterDomain, "cluster-domain", provider.DefaultKubeClusterDomain, "cluster domain")
	flags.StringVar(&c.NodeName, "nodename", hostname, "value to be set as the Node name and label node.k8s.io/hostname")
	flags.StringVar(&c.ListenAddress, "addr", provider.DefaultListenAddr, "address to bind for serving requests from the Kubernetes API server")
	flags.StringVar(&c.MetricsAddr, "metrics-addr", provider.DefaultMetricsAddr, "address to listen for metrics/stats requests")
	flags.IntVar(&c.PodSyncWorkers, "pod-sync-workers", provider.DefaultPodSyncWorkers, `number of pod synchronization workers`)
	flags.DurationVar(&c.InformerResyncPeriod, "full-resync-period", provider.DefaultInformerResyncPeriod, "interval period for recurring listing of all Pods assigned to this Node")
	flags.DurationVar(&c.StreamIdleTimeout, "stream-idle-timeout", provider.DefaultStreamIdleTimeout,
		"maximum time a streaming connection can be idle before the connection is automatically closed")
	flags.DurationVar(&c.StreamCreationTimeout, "stream-creation-timeout", provider.DefaultStreamCreationTimeout,
		"stream-creation-timeout is the maximum time for streaming connection")
	flags.StringVar(&c.ServerCertPath, "tls-cert", "", "path to the certificate to secure the kubelet API")
	flags.StringVar(&c.ServerKeyPath, "tls-key", "", "path to the private key to sign the kubelet API")
	flags.IPVar(&c.NodeInternalIP, "internal-ip", net.IPv4zero, "IP address to advertise as Node InternalIP, 0.0.0.0 means auto-detect")
	flags.IPVar(&c.NodeExternalIP, "external-ip", net.IPv4zero, "IP address to advertise as Node ExternalIP, 0.0.0.0 means auto-detect")
	flags.StringSliceVarP(&c.AllowedHostPaths, "dir", "d", provider.DefaultAllowedPaths, "only allow mounts below these directories")
	flags.BoolVarP(&c.UserMode, "user", "u", false, "rely on the user's systemd")
	flags.BoolVarP(&c.DisableTaint, "disable-taint", "", false, "disable the node taint")

	// Since klog is the logger implementation, install its flags.
	// But prepend "klog." to the flag name for clear separation.
	flagset := flag.NewFlagSet("klog", flag.PanicOnError)
	klog.InitFlags(flagset)
	flagset.VisitAll(func(f *flag.Flag) {
		f.Name = "klog." + f.Name
		flags.AddGoFlag(f)
	})
}
