package packages

import "testing"

func TestClean(t *testing.T) {
	pkg := Clean("https://www.example.org/coredns_1.7.1-be09f473-0~20.040_amd64.deb")
	if pkg != "coredns" {
		t.Fatalf("expected %s, got %s", "coredns", pkg)
	}
	pkg = Clean("/usr/bin/coredns")
	if pkg != "coredns" {
		t.Fatalf("expected %s, got %s", "coredns", pkg)
	}
}
