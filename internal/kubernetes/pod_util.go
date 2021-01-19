package kubernetes

import (
	"fmt"
	"io/ioutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/scheme"
)

// PodFromFile builds a Pod from a local file.
func PodFromFile(path string) (*corev1.Pod, error) {
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
