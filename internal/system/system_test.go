package system

import (
	"strconv"
	"testing"
)

func TestMemory(t *testing.T) {
	mem := Memory()
	if mem == "" {
		t.Fatal("expected memory size, got nothing")
	}
	// check for unit
	_, err := strconv.Atoi(mem[:len(mem)-1])
	if err != nil {
		t.Fatal(err)
	}
}

func TestPid(t *testing.T) {
	pids := Pid()
	if pids == "" {
		t.Fatal("expected pids to return number, got empty string")
	}
}

func TestImage(t *testing.T) {
	var tests = []struct {
		osReleaseFilePath, expected string
	}{
		{
			osReleaseFilePath: "testdata/os-release-rhel77",
			expected:          "Red Hat Enterprise Linux Server 7.7 (Maipo)",
		},
		{
			osReleaseFilePath: "testdata/os-release-ubuntu2004",
			expected:          "Ubuntu 20.04.1 LTS",
		},
	}

	for _, test := range tests {
		osReleaseFilePath = test.osReleaseFilePath
		actual := Image()
		if test.expected != actual {
			t.Fatalf("expected: %q, got :%q", test.expected, actual)
		}
	}
}

func TestID(t *testing.T) {
	var tests = []struct {
		osReleaseFilePath, expected string
	}{
		{
			osReleaseFilePath: "testdata/os-release-rhel77",
			expected:          "rhel",
		},
		{
			osReleaseFilePath: "testdata/os-release-ubuntu2004",
			expected:          "ubuntu",
		},
	}

	for _, test := range tests {
		osReleaseFilePath = test.osReleaseFilePath
		actual := ID()
		if test.expected != actual {
			t.Fatalf("expected: %q, got :%q", test.expected, actual)
		}
	}
}
