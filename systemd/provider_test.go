package systemd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/virtual-kubelet/systemk/pkg/manager"
	"github.com/virtual-kubelet/systemk/pkg/packages"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/scheme"
)

const dir = "testdata/provider"

func TestProviderPodSpecUnits(t *testing.T) {
	testFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not read %s: %q", dir, err)
	}
	p := new(P)
	p.m, _ = manager.NewTest()
	p.pkg = &packages.NoopPackageManager{}

	os.Setenv("HOSTNAME", "localhost")

	for _, f := range testFiles {
		if f.IsDir() {
			continue
		}

		if filepath.Ext(f.Name()) != ".yaml" {
			continue
		}
		base := f.Name()[:len(f.Name())-5]
		t.Run("Testing: "+base, func(t *testing.T) {
			testPodSpecUnit(t, p, base)
		})
	}
}

func testPodSpecUnit(t *testing.T, p *P, base string) {
	yamlFile := filepath.Join(dir, base+".yaml")
	pod, err := podFromFile(yamlFile)
	if err != nil {
		t.Error(err)
		return
	}
	unitFile := filepath.Join(dir, base+".units")
	unit, _ := ioutil.ReadFile(unitFile)

	if err := p.CreatePod(context.TODO(), pod); err != nil {
		t.Errorf("failed to call CreatePod: %v", err)
		return
	}
	// now it's just string compare
	got := ""
	for _, c := range pod.Spec.Containers {
		name := PodToUnitName(pod, c.Name)
		got += p.m.Unit(name)
	}

	if got != string(unit) {
		t.Errorf("got unexpected result: %s", diff.LineDiff(got, string(unit)))
	}
}

func podFromFile(path string) (*corev1.Pod, error) {
	podSpec, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't open %q, %s", path, err)
	}
	d := scheme.Codecs.UniversalDeserializer()
	obj, _, err := d.Decode(podSpec, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decode yaml %q: %s", path, err)
	}
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("%s is a not a podSpec", path)
	}
	// Set these to a fixed value.
	pod.ObjectMeta.Namespace = "default"
	pod.ObjectMeta.UID = "aa-bb"
	return pod, nil
}
