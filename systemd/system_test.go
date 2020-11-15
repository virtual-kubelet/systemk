package systemd

import (
	"strconv"
	"testing"
)

func TestMemory(t *testing.T) {
	mem := memory()
	if mem == "" {
		t.Fatal("expected memory size, got nothing")
	}
	// check for unit
	num, err := strconv.Atoi(mem[:len(mem)-1])
	if err != nil {
		t.Fatal(err)
	}
}
