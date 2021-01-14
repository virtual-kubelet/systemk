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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	cli "github.com/virtual-kubelet/node-cli"
	"github.com/virtual-kubelet/node-cli/opts"
	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/systemk/pkg/system"
	"github.com/virtual-kubelet/systemk/systemd"
	vklog "github.com/virtual-kubelet/virtual-kubelet/log"
	vkklogv2 "github.com/virtual-kubelet/virtual-kubelet/log/klogv2"
)

var (
	buildVersion = "N/A"
	buildTime    = "N/A"
	k8sVersion   = "v1.18.4" // Inject this build time by parsing mod.go
)

func main() {
	var (
		certFile string
		keyFile  string
		nodeIP   string
		nodeEIP  string
		topdirs  []string
	)

	flags := pflag.NewFlagSet("client", pflag.ContinueOnError)
	// these options need to be make inline with k3s or make clear they afffect logs, certfile and keyfile is too short
	flags.StringVar(&certFile, "certfile", "", "certfile")
	flags.StringVar(&keyFile, "keyfile", "", "keyfile")
	flags.StringVarP(&nodeIP, "node-ip", "i", "", "IP address to advertise for node, '0.0.0.0' or not provided to auto-detect")
	flags.StringVar(&nodeEIP, "node-external-ip", "", "External IP address to advertise for node, '0.0.0.0' or not provided to auto-detect")
	flags.StringSliceVarP(&topdirs, "dir", "d", []string{"/var"}, "Only allow mounts below these directories")

	ctx := cli.ContextWithCancelOnSignal(context.Background())

	vklog.L = vkklogv2.New(nil)

	o, err := opts.FromEnv()
	if err != nil {
		vklog.G(ctx).Fatal(err)
	}
	o.Provider = "systemd"
	o.Version = strings.Join([]string{k8sVersion, "systemk", buildVersion}, "-")

	node, err := cli.New(ctx,
		cli.WithBaseOpts(o),
		cli.WithPersistentFlags(flags),
		// Validate configuration after flag parsing.
		cli.WithPersistentPreRunCallback(func() error {
			// Enforce usage of Node Leases API.
			o.EnableNodeLease = true

			// Ensure Node name is set.
			//
			// Setting the Node name is computed in the following order:
			// 1. flag --nodename, or if not set
			// 2. environment variable HOSTNAME, or if not set
			// 3. the hostname of the machine where systemk is running.
			if flag := flags.Lookup("nodename"); flag != nil {
				if !flag.Changed {
					var defaultHostname string
					if defaultHostname = os.Getenv("HOSTNAME"); defaultHostname == "" {
						defaultHostname = system.Hostname()
					}
					o.NodeName = defaultHostname
				}
			}
			return nil
		}),
		cli.WithCLIVersion(buildVersion, buildTime),
		cli.WithKubernetesNodeVersion(k8sVersion),
		cli.WithProvider("systemd", func(cfg provider.InitConfig) (provider.Provider, error) {
			cfg.ConfigPath = o.KubeConfigPath
			p, err := systemd.New(ctx, cfg)
			if err != nil {
				return p, err
			}
			p.SetNodeIPs(nodeIP, nodeEIP)
			vklog.G(ctx).Infof("Using internal/external IP addresses: %s/%s", p.NodeInternalIP.Address, p.NodeExternalIP.Address)

			p.Topdirs = topdirs
			if certFile == "" || keyFile == "" {
				vklog.G(ctx).Info("No certificates found, disabling GetContainerLogs")
				return p, nil
			}

			r := mux.NewRouter()
			r.HandleFunc("/containerLogs/{namespace}/{pod}/{container}", p.GetContainerLogsHandler).Methods("GET")
			r.NotFoundHandler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			})
			go func() {
				err := http.ListenAndServeTLS(fmt.Sprintf(":%d", cfg.DaemonPort), certFile, keyFile, r)
				if err != nil {
					vklog.G(ctx).Fatal(err)
				}
			}()
			return p, nil
		}),
	)

	if err != nil {
		vklog.G(ctx).Fatal(err)
	}

	if err := node.Run(ctx); err != nil {
		vklog.G(ctx).Fatal(err)
	}
}
