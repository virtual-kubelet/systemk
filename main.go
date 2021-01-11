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
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	cli "github.com/virtual-kubelet/node-cli"
	"github.com/virtual-kubelet/node-cli/opts"
	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/systemk/pkg/system"
	"github.com/virtual-kubelet/systemk/systemd"
	"k8s.io/klog/v2"
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
	flags.StringVarP(&nodeIP, "node-ip", "i", "", "IP address to advertise for node")
	flags.StringVar(&nodeEIP, "node-external-ip", "", "External IP address to advertise for node")
	flags.StringSliceVarP(&topdirs, "dir", "d", []string{"/var"}, "Only allow mounts below these directories")

	ctx := cli.ContextWithCancelOnSignal(context.Background())

	o, err := opts.FromEnv()
	if err != nil {
		klog.Fatal(err)
	}
	o.Provider = "systemd"
	o.Version = strings.Join([]string{k8sVersion, "vk-systemd", buildVersion}, "-")
	o.NodeName = system.Hostname()
	o.EnableNodeLease = true

	node, err := cli.New(ctx,
		cli.WithBaseOpts(o),
		cli.WithPersistentFlags(flags),
		cli.WithCLIVersion(buildVersion, buildTime),
		cli.WithKubernetesNodeVersion(k8sVersion),
		cli.WithProvider("systemd", func(cfg provider.InitConfig) (provider.Provider, error) {
			cfg.ConfigPath = o.KubeConfigPath
			p, err := systemd.New(ctx, cfg)
			if err != nil {
				return p, err
			}
			p.SetNodeIPs(nodeIP, nodeEIP)
			klog.Infof("Using internal/external IP addresses: %s/%s", p.NodeInternalIP.Address, p.NodeExternalIP.Address)

			p.Topdirs = topdirs
			if certFile == "" || keyFile == "" {
				klog.Info("No certificates found, disabling GetContainerLogs")
				return p, nil
			}

			r := mux.NewRouter()
			r.HandleFunc("/containerLogs/{namespace}/{pod}/{container}", p.GetContainerLogsHandler).Methods("GET")
			r.NotFoundHandler = http.HandlerFunc(p.NotFound)
			go func() {
				err := http.ListenAndServeTLS(fmt.Sprintf(":%d", cfg.DaemonPort), certFile, keyFile, r)
				if err != nil {
					klog.Fatal(err)
				}
			}()
			return p, nil
		}),
	)

	if err != nil {
		klog.Fatal(err)
	}

	if err := node.Run(ctx); err != nil {
		klog.Fatal(err)
	}
}
