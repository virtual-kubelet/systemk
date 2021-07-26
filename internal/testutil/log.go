package testutil

import (
	"bufio"

	"github.com/virtual-kubelet/systemk/internal/provider"
	nodeapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
)

// ContainerOutput will return the last line of the log from the container.
func ContainerOutput(namespace, name, container string) (string, error) {
	sdjournal, err := provider.JournalReader(namespace, name, container, nodeapi.ContainerLogOpts{SinceSeconds: 10})
	if err != nil {
		return "", err
	}
	defer sdjournal.Close()

	scanner := bufio.NewScanner(sdjournal)
	text := ""
	for scanner.Scan() {
		text = scanner.Text()
	}
	return text, scanner.Err()
}
