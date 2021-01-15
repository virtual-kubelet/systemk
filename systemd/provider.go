package systemd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/systemk/pkg/manager"
	"github.com/virtual-kubelet/systemk/pkg/packages"
	"github.com/virtual-kubelet/systemk/pkg/system"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// unitDir is where systemk stores the modified unit files.
var unitDir = "/var/run/systemk"

// P is a systemd provider.
type P struct {
	m   manager.Manager
	pkg packages.PackageManager

	secretLister listersv1.SecretLister
	cmLister     listersv1.ConfigMapLister
	w            *watcher

	NodeInternalIP *corev1.NodeAddress
	NodeExternalIP *corev1.NodeAddress
	ClusterDomain  string
	Topdirs        []string

	nodename      string
	daemonPort    int32
	kubernetesURL string
}

// Ensure P implements updater at compile time.
var _ updater = (*P)(nil)

// New returns a new systemd provider.
func New(ctx context.Context, cfg provider.InitConfig) (*P, error) {
	if err := os.MkdirAll(unitDir, 0750); err != nil {
		return nil, err
	}
	m, err := manager.New(unitDir, false)
	if err != nil {
		return nil, err
	}
	p := &P{m: m}
	switch system.ID() {
	default:
		return nil, fmt.Errorf("unsupported system")
	case "debian", "ubuntu":
		p.pkg = new(packages.DebianPackageManager)

		// Check if we're root and otherwise skip this step.
		if os.Geteuid() != 0 {
			break
		}
		// Just installed pre-requisites instead of pointing to the docs.
		klog.Infof("Installing %s, to prevent installed daemons from starting", "policyrcd-script-zg2")
		ok, err := p.pkg.Install("policyrcd-script-zg2", "")
		if err != nil {
			klog.Warningf("Failed to install %s, %s. Continuing anyway", "policyrcd-script-zg2", err)
		}
		if ok {
			klog.Infof("%s is already installed", "policyrcd-script-zg2")
		}

	case "arch":
		p.pkg = new(packages.ArchlinuxPackageManager)
	case "noop":
		p.pkg = new(packages.NoopPackageManager)
	}

	p.ClusterDomain = cfg.KubeClusterDomain
	p.nodename = cfg.NodeName
	p.daemonPort = cfg.DaemonPort

	// Parse the kubeconfig, yet again, to gain access to the Host field,
	// which has the value to set for the KUBERNETES_SERVICE_* Pod env vars.
	if cfg.ConfigPath == "" {
		return p, nil
	}
	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.ConfigPath},
		&clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return p, err
	}
	p.kubernetesURL = restConfig.Host

	// Set-up a clientset to be used in Secret and ConfigMap reconciliation.
	clientset, err := nodeutil.ClientsetFromEnv(cfg.ConfigPath)
	if err != nil {
		return p, err
	}

	// Set the event handler functions for Secrets and ConfigMaps.
	p.w = newWatcher()

	// Set-up informers and extract listers to use when accessing Kubernetes resources.
	informerFactory := informers.NewSharedInformerFactory(clientset, time.Minute*1)

	secretInformer := informerFactory.Core().V1().Secrets()
	secretInformer.Informer().AddEventHandler(p.w.handlerFuncs(ctx, p))
	p.secretLister = secretInformer.Lister()

	cmInformer := informerFactory.Core().V1().ConfigMaps()
	cmInformer.Informer().AddEventHandler(p.w.handlerFuncs(ctx, p))
	p.cmLister = cmInformer.Lister()

	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())

	return p, nil
}

func (p *P) SetNodeIPs(nodeIP, nodeEIP string) {
	// Get the addresses.
	internal, external := nodeAddresses()
	if nodeIP != "" {
		p.NodeInternalIP = &corev1.NodeAddress{Address: nodeIP, Type: corev1.NodeInternalIP}
	} else {
		p.NodeInternalIP = internal
	}
	if nodeEIP != "" {
		p.NodeExternalIP = &corev1.NodeAddress{Address: nodeEIP, Type: corev1.NodeExternalIP}
	} else {
		p.NodeExternalIP = external
	}
	if p.NodeExternalIP == nil && p.NodeInternalIP == nil {
		klog.Fatal("Can not find internal or external IP address")
	}
	if p.NodeExternalIP == nil {
		p.NodeExternalIP = p.NodeInternalIP
	}
	if p.NodeInternalIP == nil {
		p.NodeInternalIP = p.NodeExternalIP
	}
}

var _ provider.Provider = new(P)
