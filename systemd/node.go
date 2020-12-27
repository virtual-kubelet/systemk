package systemd

import (
	"context"
	"net"

	"github.com/virtual-kubelet/systemk/pkg/system"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultOS = "Linux"
)

// ConfigureNode enables a provider to configure the node object that will be used for Kubernetes.
func (p *P) ConfigureNode(ctx context.Context, node *corev1.Node) {
	node.Status.Capacity = capacity()
	node.Status.Allocatable = capacity()
	node.Status.Conditions = nodeConditions()
	node.Status.Addresses = append([]corev1.NodeAddress{*p.NodeInternalIP}, *p.NodeExternalIP)
	node.Status.DaemonEndpoints = nodeDaemonEndpoints(p.daemonPort)
	node.Status.NodeInfo.OperatingSystem = defaultOS
	node.Status.NodeInfo.KernelVersion = system.Kernel()
	node.Status.NodeInfo.OSImage = system.Image()
	node.Status.NodeInfo.ContainerRuntimeVersion = system.Version()
	node.ObjectMeta = metav1.ObjectMeta{
		Name: p.nodename,
		Labels: map[string]string{
			"node.kubernetes.io/instance-type":  "systemk",
			"kubernetes.io/os":                  defaultOS,
			"kubernetes.io/hostname":            p.nodename,
			corev1.LabelZoneFailureDomainStable: "localhost",
			corev1.LabelZoneRegionStable:        p.nodename,
		},
	}
}

// nodeAddresses finds the internal and external address of the node (if found).
func nodeAddresses() (internal, external *corev1.NodeAddress) {
	cidrs := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	rfc1918 := make([]*net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
		_, block, _ := net.ParseCIDR(cidr)
		rfc1918[i] = block
	}

	for _, ip := range system.IPs() {
		for _, block := range rfc1918 {
			if block.Contains(ip) {
				if internal == nil {
					internal = &corev1.NodeAddress{Address: ip.String(), Type: corev1.NodeInternalIP}
				}
				continue
			}

			if external == nil {
				external = &corev1.NodeAddress{Address: ip.String(), Type: corev1.NodeExternalIP}
			}
		}
	}
	return internal, external
}

// nodeDaemonEndpoints returns NodeDaemonEndpoints for the node status within Kubernetes.
func nodeDaemonEndpoints(port int32) corev1.NodeDaemonEndpoints {
	// used for logs
	return corev1.NodeDaemonEndpoints{KubeletEndpoint: corev1.DaemonEndpoint{Port: port}}
}

// capacity returns a resource list containing the capacity limits set for Zun.
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
		{
			Type:               "Ready",
			Status:             corev1.ConditionTrue,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletReady",
			Message:            "kubelet is ready.",
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
