package provider

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/virtual-kubelet/systemk/internal/kubernetes"
	"github.com/virtual-kubelet/systemk/internal/ospkg"
	"github.com/virtual-kubelet/systemk/internal/unit"
	"k8s.io/client-go/informers"
)

const dir = "../testdata/provider"

func TestProviderPodSpecUnits(t *testing.T) {
	log = &noopLogger{}
	testFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not read %s: %q", dir, err)
	}
	p := new(p)
	p.pkgManager = &ospkg.NoopManager{}
	p.unitManager, _ = unit.NewMockManager()
	p.config = &Opts{
		NodeName:       "localhost",
		NodeInternalIP: []byte{192, 168, 1, 1},
		NodeExternalIP: []byte{172, 16, 0, 1},
	}

	p.podResourceManager = kubernetes.NewPodResourceWatcher(informers.NewSharedInformerFactory(nil, 0))

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

func testPodSpecUnit(t *testing.T, p *p, base string) {
	yamlFile := filepath.Join(dir, base+".yaml")
	pod, err := kubernetes.PodFromFile(yamlFile)
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
	got := ""
	for _, c := range pod.Spec.Containers {
		name := podToUnitName(pod, c.Name)
		got += p.unitManager.Unit(name)
	}

	if got != string(unit) {
		t.Errorf("got unexpected result: %s", diff.LineDiff(got, string(unit)))
	}
}
