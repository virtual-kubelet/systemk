package systemd

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// Where this files life is still an open question, right now we bind mount everything into place.
var (
	emptyDir     = filepath.Join(os.TempDir(), "emptydirs")
	secretDir    = filepath.Join(os.TempDir(), "secrets")
	configmapDir = filepath.Join(os.TempDir(), "configmaps")
)

// volumes inspecs the volumes and returns a maaping with the volume's Name and the directory on-disk that
// should be used for this. The on-disk structure is prepared and can be used.
func (p *P) volumes(pod *corev1.Pod) (map[string]string, error) {
	vol := make(map[string]string)
	uid := string(pod.ObjectMeta.UID)
	for _, v := range pod.Spec.Volumes {
		log.Printf("Looking at volume %q", v.Name)
		switch {

		case v.EmptyDir != nil:
			dir := filepath.Join(emptyDir, uid)
			if err := os.MkdirAll(dir, 0750); err != nil {
				log.Println(err)
				return nil, err
			}
			log.Printf("Created %q for emptyDir: %s", dir, v.Name)
			vol[v.Name] = dir

		case v.Secret != nil:
			secret, err := p.rm.GetSecret(v.Secret.SecretName, pod.Namespace)
			if v.Secret.Optional != nil && !*v.Secret.Optional && errors.IsNotFound(err) {
				return nil, fmt.Errorf("secret %s is required by pod %s and does not exist", v.Secret.SecretName, pod.Name)
			}
			if secret == nil {
				continue
			}

			dir := filepath.Join(secretDir, uid)
			if err := os.MkdirAll(dir, 0750); err != nil {
				return nil, err
			}

			log.Printf("Created %q for secret: %s", dir, v.Name)

			// secret.StringData is not handled here.
			for k, v := range secret.Data {
				path := filepath.Join(dir, k)

				log.Printf("Writing secret to path %q", path)
				err := ioutil.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(v)), 0640)
				if err != nil {
					return vol, err
				}
			}
			vol[v.Name] = dir

		case v.ConfigMap != nil:
			configMap, err := p.rm.GetConfigMap(v.ConfigMap.Name, pod.Namespace)
			if v.ConfigMap.Optional != nil && !*v.ConfigMap.Optional && errors.IsNotFound(err) {
				return nil, fmt.Errorf("configMap %s is required by pod %s and does not exist", v.ConfigMap.Name, pod.Name)
			}
			if configMap == nil {
				continue
			}

			dir := filepath.Join(configmapDir, uid)
			if err := os.MkdirAll(dir, 0750); err != nil {
				return nil, err
			}
			log.Printf("Created %q for configmap: %s", dir, v.Name)

			for k, v := range configMap.Data {
				path := filepath.Join(dir, k)
				log.Printf("Writing configMap Data to path %q", path)
				err := ioutil.WriteFile(path, []byte(base64.StdEncoding.EncodeToString([]byte(v))), 0640)
				if err != nil {
					return vol, err
				}
			}
			for k, v := range configMap.BinaryData {
				path := filepath.Join(dir, k)
				log.Printf("Writing configMap BinaryData to path %q", path)
				err := ioutil.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(v)), 0640)
				if err != nil {
					return vol, err
				}
			}
			vol[v.Name] = dir

		default:
			return nil, fmt.Errorf("pod %s requires volume %s which is of an unsupported type", pod.Name, v.Name)
		}
	}

	return vol, nil
}
