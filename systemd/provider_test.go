package systemd

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/miekg/systemk/pkg/manager"
	"github.com/miekg/systemk/pkg/packages"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const dir = "testdata"

func TestProviderPodSpecUnits(t *testing.T) {
	testFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not read %s: %q", dir, err)
	}
	p := new(P)
	p.m, _ = manager.NewTest()
	p.pkg = &packages.TestPackageManager{}

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
	podSpec, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		t.Errorf("couldn't open %q, %s", yamlFile, err)
		return
	}
	d := scheme.Codecs.UniversalDeserializer()
	obj, _, err := d.Decode(podSpec, nil, nil)
	if err != nil {
		log.Fatalf("could not decode yaml %q: %s", yamlFile, err)
	}
	unitFile := filepath.Join(dir, base+".units")
	unit, _ := ioutil.ReadFile(unitFile)

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		t.Errorf("%s is a not a podSpec", yamlFile)
		return
	}
	pod.ObjectMeta.Namespace = "default"
	pod.ObjectMeta.UID = "aa-bb"

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
