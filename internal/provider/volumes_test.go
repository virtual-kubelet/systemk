package provider

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

func TestIsBelowPath(t *testing.T) {
	tests := []struct {
		top   string
		path  string
		below bool
	}{
		{"/", "/tmp/x", true},
		{"/", "/", false},
		{"/tmp", "/", false},
		{"/tmp/x", "/", false},
	}
	for i, tc := range tests {
		ok := isBelowPath(tc.top, tc.path)
		if ok != tc.below {
			t.Errorf("test %d, expected %t, got %t", i, tc.below, ok)
		}
	}
}
func TestIsBelow(t *testing.T) {
	if err := isBelow([]string{"/var", "/tmp"}, "/tmp/x"); err != nil {
		t.Errorf("/tmp/x should be below /tmp")
	}

	if err := isBelow([]string{"/var", "/tmp"}, "/"); err == nil {
		t.Errorf("/ should not be below /tmp or /var")
	}

	if err := isBelow([]string{"/var", "/tmp"}, "/var"); err == nil {
		t.Errorf("/var should not be below /tmp or /var")
	}
}
