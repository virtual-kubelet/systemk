package testutil

import (
	"io"
	"os/exec"
)

// We're using 'k3s kubectl' for our testing

// KubeApply runs 'kubectl apply -f' with the data from r.
func KubeApply(r io.Reader) ([]byte, error) {
	cmd := exec.Command("k3s", "kubectl", "apply", "-f", "-")
	cmd.Stdin = r

	return cmd.Output()
}

// KubeDelete runs 'kubectl delete -f' with the data from r.
func KubeDelete(r io.Reader) ([]byte, error) {
	cmd := exec.Command("k3s", "kubectl", "delete", "-f", "-")
	cmd.Stdin = r

	return cmd.Output()
}

// KubeWait will wait for pod n to be ready
func KubeWait(n string) ([]byte, error) {
	cmd := exec.Command("k3s", "kubectl", "wait", "--for=condition=ready", "--timeout", "30s", "pod", n)
	// use labels here in the future?
	return cmd.Output()
}
