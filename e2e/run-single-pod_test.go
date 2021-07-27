// +build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/virtual-kubelet/systemk/internal/testutil"
)

func TestRunSinglePod(t *testing.T) {
	const yaml = `
apiVersion: v1
kind: Pod
metadata:
  name: uptimed
  labels:
    name: uptimed
spec:
  securityContext:
    runAsUser: 0
  containers:
  - name: uptimed
    image: uptimed
`
	if out, err := testutil.KubeApply(strings.NewReader(yaml)); err != nil {
		t.Fatalf("fail: %s: %s", err, out)
	}

	if out, err := testutil.KubeWait("uptimed"); err != nil {
		t.Fatalf("fail: %s: %s", err, out)
	}

	if out, err := testutil.KubeDelete(strings.NewReader(yaml)); err != nil {
		t.Fatalf("fail: %s: %s", err, out)
	}

}
