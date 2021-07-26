// +build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/virtual-kubelet/systemk/internal/testutil"
)

func TestInitContainerEmptyDirWrite(t *testing.T) {
	const yaml = `
apiVersion: v1
kind: Pod
metadata:
  name: uptimed
  labels:
    name: uptimed
spec:
  securityContext:
    runAsUser: 1
  initContainers:
  - name: init-uptimed
    image: bash
    command: ["bash", "-c"]
    args: ["echo goodbye > /etc/uptimed/hello"]
    volumeMounts:
    - mountPath: /etc/uptimed
      name: shared-dir
  containers:
  - name: uptimed
    image: uptimed
  - name: cat
    image: /bin/bash
    command: ["bash", "-c"]
    args: ["cat /etc/uptimed/hello"]
    volumeMounts:
    - mountPath: /etc/uptimed
      name: shared-dir
  volumes:
  - name: shared-dir
    emptyDir: {}
`

	if out, err := testutil.KubeApply(strings.NewReader(yaml)); err != nil {
		t.Fatalf("fail: %s: %s", err, out)
	}

	if out, err := testutil.KubeWait("uptimed"); err != nil {
		t.Fatalf("fail: %s: %s", err, out)
	}

	line, err := testutil.ContainerOutput("default", "uptimed", "cat")
	if err != nil {
		t.Fatalf("fail: %s", err)
	}

	if line == "goodbye\n" {
		t.Errorf("expected %s, got %s", "goodbye\n", line)
	}

	if out, err := testutil.KubeDelete(strings.NewReader(yaml)); err != nil {
		t.Fatalf("fail: %s: %s", err, out)
	}
}
