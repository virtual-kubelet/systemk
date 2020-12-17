package systemd

import (
	"testing"
)

func TestWatchUnWatch(t *testing.T) {
	w := newWatcher(nil)
	pod, err := podFromFile("testdata/uptimed-config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	w.Watch(pod)
	if len(w.configs) != 1 {
		t.Fatal("expected 1 configMap to be watched")
	}
	pods := w.configs["default.uptimed-conf"] // namespace . configMap name
	if len(pods) != 1 {
		t.Fatal("expected pod to be returned for watch")
	}

	w.Unwatch(pods[0]) // call unwatch with the returned pod, so we test the correct pod has been stored
	if len(w.configs) != 0 {
		t.Fatal("expected no configMaps to be watched")
	}
}
