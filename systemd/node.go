package systemd

import (
	"context"
	"net"

	"github.com/virtual-kubelet/systemk/pkg/system"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const(
	defaultOS="Linux"
)

// ConfigureNode enables a provider to configure the node object that will be used for Kubernetes.
func (p *P) ConfigureNode(ctx context.Context, node *corev1.Node) {
	node.Status.Capacity = p.capacity()
	node.Status.Allocatable = p.capacity()
	node.Status.Conditions = p.nodeConditions()
	node.Status.Addresses = p.nodeAddresses()
	node.Status.DaemonEndpoints = p.nodeDaemonEndpoints()
	node.Status.NodeInfo.OperatingSystem = defaultOS
	node.Status.NodeInfo.KernelVersion = system.Kernel()
	node.Status.NodeInfo.OSImage = system.Image()
	node.Status.NodeInfo.ContainerRuntimeVersion = system.Version()
	node.ObjectMeta = metav1.ObjectMeta{
		Name: system.Hostname(),
		Labels: map[string]string{
			"type":                              "virtual-kubelet",
			"kubernetes.io/os":                  defaultOS,
			"kubernetes.io/hostname":            system.Hostname(),
			corev1.LabelZoneFailureDomainStable: "localhost",
			corev1.LabelZoneRegionStable:        system.Hostname(),
		},
	}
	p.Addresses = node.Status.Addresses
}

// nodeAddresses returns a list of addresses for the node status within Kubernetes.
func (p *P) nodeAddresses() []corev1.NodeAddress {
	cidrs := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	rfc1918 := make([]*net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
		_, block, _ := net.ParseCIDR(cidr)
		rfc1918[i] = block
	}

	ips := system.IPs()
	na := make([]corev1.NodeAddress, len(ips))
	for i, ip := range ips {
		for _, block := range rfc1918 {
			na[i] = corev1.NodeAddress{
				Address: ip.String(),
				Type:    corev1.NodeExternalIP,
			}
			if block.Contains(ip) {
				na[i].Type = corev1.NodeInternalIP
			}
		}
	}
	// corev1.NodeInternalDNS??
	return na
}

// nodeDaemonEndpoints returns NodeDaemonEndpoints for the node status within Kubernetes.
func (p *P) nodeDaemonEndpoints() corev1.NodeDaemonEndpoints {
	// use for logs
	return corev1.NodeDaemonEndpoints{
		KubeletEndpoint: corev1.DaemonEndpoint{
			Port: p.DaemonPort,
		},
	}
}

// capacity returns a resource list containing the capacity limits set for Zun.
func (p *P) capacity() corev1.ResourceList {
	return corev1.ResourceList{
		"cpu":     resource.MustParse(system.CPU()),
		"memory":  resource.MustParse(system.Memory()),
		"pods":    resource.MustParse(system.Pid()),
		"storage": resource.MustParse("40G"), // We're using tmpfs _a_ lot.. so
	}
}

func (p *P) nodeConditions() []corev1.NodeCondition {
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

// externalOrInternalAddress prefers to return the external address of the node, if not available
// an internal address will be returned. If neither is found a 127.0.0.1 node address is synthesized.
func externalOrInternalAddress(addrs []corev1.NodeAddress) corev1.NodeAddress {
	for _, a := range addrs {
		if a.Type == corev1.NodeExternalIP {
			return a
		}
	}
	for _, a := range addrs {
		if a.Type == corev1.NodeInternalIP {
			return a
		}
	}
	return corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: "127.0.0.1"}
}
