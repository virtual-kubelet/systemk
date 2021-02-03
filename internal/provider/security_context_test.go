package provider

import (
	"testing"

	"github.com/virtual-kubelet/systemk/internal/kubernetes"
)

func TestUidGidFromSecurityContext(t *testing.T) {
	var tests = []struct {
		file    string
		rootmap [2]int
		uid     [2]string
		gid     [2]string
	}{
		{
			// asks for runAsUser: 0, with no mapping expect 0, with mapping to 1, expect 1
			"../testdata/uptimed-config.yaml",
			[2]int{0, 1},
			[2]string{"0", "1"},
			[2]string{"0", "1"},
		},
		{
			// asks for runAsUserName: "root", with no mapping expect 0, with mapping to 1, expect 1
			"../testdata/uptimed-runAsUserName.yaml",
			[2]int{0, 1},
			[2]string{"0", "1"},
			[2]string{"0", "1"},
		},
		{
			// no security context, even with mapping this should return empty strings
			"../testdata/uptimed-no-security-context.yaml",
			[2]int{0, 1},
			[2]string{"", ""},
			[2]string{"", ""},
		},
	}

	for i, tc := range tests {
		pod, err := kubernetes.PodFromFile(tc.file)
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 2; j++ {
			uid, gid, err := uidGidFromSecurityContext(pod, tc.rootmap[j])
			if err != nil {
				t.Fatal(err)
			}
			if uid != tc.uid[j] {
				t.Errorf("test %d, expected uid to be %q, got %q", i, tc.uid[j], uid)
			}
			if gid != tc.gid[j] {
				t.Errorf("test %d, expected gid to be %q, got %q", i, tc.gid[j], gid)
			}
		}
	}
}
