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
	"log"
	"strings"

	"github.com/miekg/vks/pkg/system"
	"github.com/miekg/vks/systemd"
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
		cli.WithCLIVersion(buildVersion, buildTime),
		cli.WithKubernetesNodeVersion(k8sVersion),
		cli.WithProvider("systemd", func(_ provider.InitConfig) (provider.Provider, error) {
			return systemd.NewProvider()
		}),
	)

	if err != nil {
		log.Fatal(err)
	}

	if err := node.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
