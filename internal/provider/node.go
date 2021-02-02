package provider

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/v22/util"
	"github.com/virtual-kubelet/systemk/internal/system"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigureNode builds the Node to be registered.
func (p *p) ConfigureNode(ctx context.Context, opts *Opts) (*v1.Node, error) {
	// This should be safe, given the address has been validated before.
	addr := strings.Split(opts.ListenAddress, ":")
	daemonPort, _ := strconv.Atoi(addr[1])

	nodeAddresses, err := computeNodeAddresses(opts)
	if err != nil {
		// TODO(pires) wrap the error in a new more meaningful error.
		return nil, err
	}

	machineID, _ := util.GetMachineID()
	taints := []v1.Taint{}
	if !opts.DisableTaint {
		taints = []v1.Taint{
			{
				Key:    DefaultTaintKey,
				Value:  DefaultTaintValue,
				Effect: corev1.TaintEffectNoSchedule,
			},
		}
	}

	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.NodeName,
			Labels: map[string]string{
				corev1.LabelInstanceTypeStable: "systemk",
				corev1.LabelOSStable:           DefaultOperatingSystem,
				corev1.LabelHostname:           opts.NodeName,
				corev1.LabelArchStable:         runtime.GOARCH,
			},
		},
		Spec: v1.NodeSpec{
			Taints: taints,
		},
		Status: v1.NodeStatus{
			Addresses:   nodeAddresses,
			Allocatable: capacity(),
			Capacity:    capacity(),
			Conditions:  nodeConditions(),
			// TODO(pires) port may be 0, which means it will be determined during runtime
			DaemonEndpoints: corev1.NodeDaemonEndpoints{KubeletEndpoint: corev1.DaemonEndpoint{Port: int32(daemonPort)}},
			NodeInfo: v1.NodeSystemInfo{
				Architecture:            runtime.GOARCH,
				ContainerRuntimeVersion: system.Version(),
				KernelVersion:           system.Kernel(),
				KubeletVersion:          opts.Version,
				MachineID:               machineID,
				OperatingSystem:         DefaultOperatingSystem,
				OSImage:                 system.Image(),
			},
		},
	}, nil
}

// computeNodeAddresses finds the internal and external address of the node (if found).
func computeNodeAddresses(config *Opts) ([]corev1.NodeAddress, error) {
	// Profile CIDRs reserved for private subnets.
	// This is useful when auto-detecting IPs and segregating which are private.
	cidrs := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	rfc1918 := make([]*net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
		_, block, _ := net.ParseCIDR(cidr)
		rfc1918[i] = block
	}
	// Try and auto-detect node addresses.
	var internalIPs, externalIPs []net.IP
	for _, ip := range system.IPs() {
		for _, block := range rfc1918 {
			// Is internal IP?
			if block.Contains(ip) {
				internalIPs = append(internalIPs, ip)
			} else {
				externalIPs = append(externalIPs, ip)
			}
		}
	}

	var nodeAddresses []corev1.NodeAddress
	var errs []error

	// If specified IP is 0.0.0.0, return auto-detected internal IPs.
	if config.NodeInternalIP.IsUnspecified() {
		if len(internalIPs) == 0 {
			errs = append(errs, fmt.Errorf("failed to auto-detect internal IP"))
		} else {
			for _, ip := range internalIPs {
				nodeAddresses = append(nodeAddresses, corev1.NodeAddress{Address: ip.String(), Type: corev1.NodeInternalIP})
				config.NodeInternalIP = ip
				// TODO(pires) check the argument below with sig-network.
				// Exit loop early, because it seems only IP of this type is supported.
				break
			}
		}
		// Otherwise, return pre-defined internal IP.
	} else {
		nodeAddresses = append(nodeAddresses, corev1.NodeAddress{Address: config.NodeInternalIP.String(), Type: corev1.NodeInternalIP})
	}

	// If specified IP is 0.0.0.0, return auto-detected external IPs.
	if config.NodeExternalIP.IsUnspecified() {
		if len(externalIPs) == 0 {
			errs = append(errs, fmt.Errorf("failed to auto-detect external IP"))
		} else {
			for _, ip := range externalIPs {
				nodeAddresses = append(nodeAddresses, corev1.NodeAddress{Address: ip.String(), Type: corev1.NodeExternalIP})
				config.NodeExternalIP = ip
				// TODO(pires) check the argument below with sig-network.
				// Exit loop early, because it seems only IP of this type is supported.
				break
			}
		}
		// Otherwise, return pre-defined external IP.
	} else {
		nodeAddresses = append(nodeAddresses, corev1.NodeAddress{Address: config.NodeExternalIP.String(), Type: corev1.NodeExternalIP})
	}

	// TODO(pires) ignore errs for now.
	return nodeAddresses, nil
}

// capacity returns a resource list containing the identifiable system resources.
func capacity() corev1.ResourceList {
	return corev1.ResourceList{
		"cpu":     resource.MustParse(system.CPU()),
		"memory":  resource.MustParse(system.Memory()),
		"pods":    resource.MustParse(system.Pid()),
		"storage": resource.MustParse("40G"), // needs the size of the /var FS
	}
}

func nodeConditions() []corev1.NodeCondition {
	return []corev1.NodeCondition{
		// Don't set the Ready condition to True just yet.
		// That shall happen when all of systemk has been initialized.
		// That code is in cmd/root.go, just after Node and Pod controllers are running.
		{
			Type:               "Ready",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletNotReady",
			Message:            "kubelet is not ready.",
		},
		{
			Type:               "OutOfDisk",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientDisk",
			Message:            "kubelet has sufficient disk space available",
		},
		{
			Type:               "MemoryPressure",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientMemory",
			Message:            "kubelet has sufficient memory available",
		},
		{
			Type:               "DiskPressure",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasNoDiskPressure",
			Message:            "kubelet has no disk pressure",
		},
		{
			Type:               "NetworkUnavailable",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "RouteCreated",
			Message:            "RouteController created a route",
		},
		{
			Type:               "PIDPressure",
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientPIDs",
			Message:            "kubelet has sufficient PIDs available",
		},
	}
}
