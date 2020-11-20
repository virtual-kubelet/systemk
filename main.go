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
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/miekg/vks/pkg/system"
	"github.com/miekg/vks/systemd"
	"github.com/spf13/pflag"
	cli "github.com/virtual-kubelet/node-cli"
	"github.com/virtual-kubelet/node-cli/opts"
	"github.com/virtual-kubelet/node-cli/provider"
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
	)
	flags := pflag.NewFlagSet("client", pflag.ContinueOnError)
	flags.StringVar(&certFile, "certfile", "", "certfile")
	flags.StringVar(&keyFile, "keyfile", "", "keyfile")

	ctx := cli.ContextWithCancelOnSignal(context.Background())

	o, err := opts.FromEnv()
	if err != nil {
		log.Fatal(err)
	}
	o.Provider = "systemd"
	o.Version = strings.Join([]string{k8sVersion, "vk-systemd", buildVersion}, "-")
	o.NodeName = system.Hostname()

	node, err := cli.New(ctx,
		cli.WithBaseOpts(o),
		cli.WithPersistentFlags(flags),
		cli.WithCLIVersion(buildVersion, buildTime),
		cli.WithKubernetesNodeVersion(k8sVersion),
		cli.WithProvider("systemd", func(cfg provider.InitConfig) (provider.Provider, error) {
			logs := true
			if certFile == "" || keyFile == "" {
				logs = false
				log.Printf("Not certificates found, disabling GetContainerLogs")
			}
			p, err := systemd.New(cfg)
			if err != nil {
				return p, err
			}

			if logs {
				r := mux.NewRouter()
				r.HandleFunc("/containerLogs/{namespace}/{pod}/{container}", p.GetContainerLogsHandler).Methods("GET")
				r.NotFoundHandler = http.HandlerFunc(p.NotFound)
				go func() {
					err := http.ListenAndServeTLS(fmt.Sprintf(":%d", cfg.DaemonPort), certFile, keyFile, r)
					if err != nil {
						log.Fatal(err)
					}
				}()
			}
			return p, nil
		}),
	)

	if err != nil {
		log.Fatal(err)
	}

	if err := node.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
