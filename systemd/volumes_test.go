package systemd

import (
	"os"
	"testing"
)

func TestMkdirAll(t *testing.T) {
	const dir = "/tmp/bliep/bla"
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)
	d, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !d.IsDir() {
		t.Fatal("expected directory, got something else")
	}
	// do again
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
}
