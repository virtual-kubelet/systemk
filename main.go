// Copyright © 2021 The systemk authors
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
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/coreos/go-systemd/v22/util"
	"github.com/pkg/errors"
	"github.com/virtual-kubelet/systemk/cmd"
	"github.com/virtual-kubelet/systemk/internal/provider"
	vklog "github.com/virtual-kubelet/virtual-kubelet/log"
	vklogv2 "github.com/virtual-kubelet/virtual-kubelet/log/klogv2"
)

var (
	buildVersion = "N/A"
	buildTime    = "N/A"
	k8sVersion   = "v1.18.15" // This should follow the version of k8s.io/kubernetes we are importing
)

func init() {
	// Fail fast if systemd is not running.
	if !util.IsRunningSystemd() {
		panic("systemd is not running, quitting")
	}
}

var log = vklogv2.New(nil)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	// Setup VK logger.
	vklog.L = vklogv2.New(map[string]interface{}{"source": "virtual-kubelet"})

	// Default systemk provider configuration.
	var opts provider.Opts
	provider.SetDefaultOpts(&opts)
	// The Kubernetes version systemk tracks.
	// This is important because of Kubernetes version skew policy.
	// See https://kubernetes.io/docs/setup/release/version-skew-policy/#kubelet
	opts.Version = k8sVersion

	// Setup the root systemk CLI command.
	rootCmd := cmd.NewRootCommand(ctx, filepath.Base(os.Args[0]), &opts)
	rootCmd.AddCommand(cmd.NewVersionCommand(buildVersion, buildTime))
	// And fire up engines!
	if err := rootCmd.Execute(); err != nil && errors.Cause(err) != context.Canceled {
		log.Fatal(err)
	}

	log.Info("system exited gracefully")

}
