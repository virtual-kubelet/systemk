package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/virtual-kubelet/systemk/internal/kubernetes"
	"github.com/virtual-kubelet/systemk/internal/ospkg"
	"github.com/virtual-kubelet/systemk/internal/system"
	"github.com/virtual-kubelet/systemk/internal/unit"
	vklogv2 "github.com/virtual-kubelet/virtual-kubelet/log/klogv2"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
)

// log is the global logger for the provider.
var log = vklogv2.New(nil)

// Provider contains the methods required to implement a virtual-kubelet provider.
//
// Errors produced by these methods should implement an interface from
// github.com/virtual-kubelet/virtual-kubelet/errdefs package in order for the
// core logic to be able to understand the type of failure.
type Provider interface {
	node.PodLifecycleHandler

	kubernetes.ResourceUpdater

	// GetContainerLogsHandler handles a Pod's container log retrieval.
	GetContainerLogsHandler(w http.ResponseWriter, r *http.Request)

	// RunInContainer executes a command in a container in the pod, copying data
	// between in/out/err and the container's stdin/stdout/stderr.
	RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error

	// ConfigureNode enables a provider to configure the Node object that
	// will be used for Kubernetes.
	ConfigureNode(context.Context, *Opts) (*corev1.Node, error)
}

// TODO(pires) uncomment when VK supports k8s.io/kubelet/pkg/apis
// PodMetricsProvider is an optional interface that providers can implement to expose pod stats
//type PodMetricsProvider interface {
//	GetStatsSummary(context.Context) (*stats.Summary, error)
//}

// p is a systemd provider.
type p struct {
	config      *Opts
	pkgManager  ospkg.Manager
	unitManager unit.Manager
	runDir      string // writeable directory under /var/run for configmap/secrets/emptydir mounts

	podResourceManager kubernetes.PodResourceManager
	kubernetesURL      string // TODO(pires) pass this in Opts
}

// Ensure p implements provider.Provider.
var _ Provider = (*p)(nil)

const defaultUnitDir = "/var/run/systemk"

// unitDir is where systemk stores the modified unit files.
// Its value changes if systemk is running in user-mode.
var unitDir = defaultUnitDir

// New returns a new systemd provider.
// informerFactory is the basis for ConfigMap and Secret retrieval and event handling.
func New(ctx context.Context, config *Opts, podWatcher kubernetes.PodResourceManager) (Provider, error) {
	// If running in user-mode, set different folder for storing unit files.
	runDir := "/var/run"
	if config.UserMode {
		unitDir = fmt.Sprintf("/var/run/user/%d/systemk", os.Geteuid())
		runDir = fmt.Sprintf("/var/run/user/%d", os.Geteuid())
	}
	if err := os.MkdirAll(unitDir, 0750); err != nil {
		return nil, err
	}
	unitManager, err := unit.NewManager(unitDir, config.UserMode)
	if err != nil {
		return nil, err
	}
	p := &p{
		unitManager:        unitManager,
		config:             config,
		podResourceManager: podWatcher,
		runDir:             runDir,
	}

	systemID := system.ID()
	switch systemID {
	case "debian", "ubuntu":
		p.pkgManager = new(ospkg.DebianManager)

		// Check if we're root and otherwise skip this step.
		if os.Geteuid() != 0 {
			break
		}
		// Just installed pre-requisites instead of pointing to the docs.
		log.Infof("installing %s, to prevent installed daemons from starting", "policyrcd-script-zg2")
		ok, err := p.pkgManager.Install("policyrcd-script-zg2", "")
		if err != nil {
			log.Warnf("failed to install %s, %s, continuing anyway", "policyrcd-script-zg2", err)
		}
		if ok {
			log.Infof("%s is already installed", "policyrcd-script-zg2")
		}
	case "arch":
		p.pkgManager = new(ospkg.ArchLinuxManager)
	default:
		log.Warnf("found unsupported package manager in %q, limiting systemk to running existing binaries", systemID)
		p.pkgManager = new(ospkg.NoopManager)
	}

	return p, nil
}
