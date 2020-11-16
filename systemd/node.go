package systemd

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigureNode enables a provider to configure the node object that will be used for Kubernetes.
func (p *P) ConfigureNode(ctx context.Context, node *corev1.Node) {
	// POD CIDR??
	node.Status.Capacity = p.capacity()
	node.Status.Allocatable = p.capacity()
	node.Status.Conditions = p.nodeConditions()
	node.Status.Addresses = p.nodeAddresses()
	node.Status.DaemonEndpoints = p.nodeDaemonEndpoints()
	node.Status.NodeInfo.OperatingSystem = "Linux"
	node.Status.NodeInfo.KernelVersion = kernel()
	node.Status.NodeInfo.OSImage = image()
	node.Status.NodeInfo.ContainerRuntimeVersion = version()
	node.ObjectMeta = metav1.ObjectMeta{
		Name: hostname(),
		Labels: map[string]string{
			"type":                   "virtual-kubelet",
			"kubernetes.io/role":     "agent",
			"kubernetes.io/hostname": hostname(),
		},
	}
}

// nodeAddresses returns a list of addresses for the node status within Kubernetes.
func (p *P) nodeAddresses() []corev1.NodeAddress { return nil }

// nodeDaemonEndpoints returns NodeDaemonEndpoints for the node status within Kubernetes.
func (p *P) nodeDaemonEndpoints() corev1.NodeDaemonEndpoints {
	return corev1.NodeDaemonEndpoints{
		KubeletEndpoint: corev1.DaemonEndpoint{
			Port: 10, /* p.daemonEndpointPort,*/
		},
	}
}

// capacity returns a resource list containing the capacity limits set for Zun.
func (p *P) capacity() corev1.ResourceList {
	return corev1.ResourceList{
		"cpu":     resource.MustParse(cpu()),
		"memory":  resource.MustParse(memory()),
		"pods":    resource.MustParse("110"), // entire PID space??
		"storage": resource.MustParse("40G"),
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
