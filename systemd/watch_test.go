package systemd

import (
	"testing"

	"k8s.io/apimachinery/pkg/types"
)

func TestWatcher(t *testing.T) {
	w := newWatcher()

	if len(w.configs) != 0 {
		t.Fatal("expected no configMaps to be watched")
	}
	if len(w.cmKeysByPod) != 0 {
		t.Fatal("expected no configMaps to be watched")
	}

	pod, err := podFromFile("testdata/uptimed-config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	w.Watch(pod)

	if len(w.configs) != 1 {
		t.Fatal("expected 1 configMap to be watched")
	}
	var configMapKey types.NamespacedName
	for k, _ := range w.configs {
		configMapKey = k
	}

	pods, _ := w.configs[configMapKey]
	if len(pods) != 1 {
		t.Fatal("expected one pod to be known to the watcher")
	}

	if len(w.cmKeysByPod) != 1 {
		t.Fatal("expected one pod to be known to the watcher")
	}
	podKey := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	if _, ok := w.cmKeysByPod[podKey]; !ok {
		t.Fatalf("expected pod %q to be known to the watcher", podKey)
	}

	w.Unwatch(pod)

	if len(w.configs) != 0 {
		t.Fatal("expected no configMaps to be watched")
	}
	if len(w.cmKeysByPod) != 0 {
		t.Fatal("expected no configMaps to be watched")
	}

	pods, _ = w.configs[configMapKey]
	if len(pods) != 0 {
		t.Fatal("expected no pods to be known to the watcher")
	}
}
