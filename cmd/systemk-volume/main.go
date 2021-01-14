package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

/*
Make this fairly simple, a bunch of global options and then a command.
We don't need per-command options, so it will just be:
systemk-volume -p UID clean|chown [UID/GID]|whatever

Note we do allow parameters to exist for each command
*/

// Defined commands.
const (
	clean = "clean" // clean up /var/run for this pod.
)

func main() {
	var (
		poduid string
	)
	pflag.StringVarP(&poduid, "poduid", "p", "", "the pod UID")
	pflag.Parse()

	if poduid == "" {
		klog.Error("Poduid must be specified")
		os.Exit(1)
	}

	if strings.Contains(poduid, "..") {
		klog.Errorf("poduid can not contain %q: %s", "..", poduid)
		os.Exit(1)
	}

	args := pflag.Args()
	if len(args) == 0 {
		klog.Error("Expecting a command")
		os.Exit(1)
	}
	var err error
	switch args[0] {
	case clean:
		err = doClean(poduid)
	default:
		klog.Errorf("Unknown command: %q", args[0])
		os.Exit(1)
	}
	if err != nil {
		klog.Error("Failed to perform %q: %s", args[0], err)
		os.Exit(1)
	}
}

func doClean(poduid string) error {
	p = path.Clean(p)
	podEphemeralVolumes := filepath.Join("/var/run", poduid)
	return os.RemoveAll(podEphemeralVolumes)
}
