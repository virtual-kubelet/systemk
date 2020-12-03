package systemd

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// Where this files life is still an open question, right now we bind mount everything into place.
var (
	varrun       = "/var/run"
	emptyDir     = "emptydirs"
	secretDir    = "secrets"
	configmapDir = "configmaps"
)

// volumes inspecs the volumes and returns a maaping with the volume's Name and the directory on-disk that
// should be used for this. The on-disk structure is prepared and can be used.
func (p *P) volumes(pod *corev1.Pod) (map[string]string, error) {
	vol := make(map[string]string)
	id := string(pod.ObjectMeta.UID)
	uid, gid := UidGidFromSecurityContext(pod)
	for i, v := range pod.Spec.Volumes {
		log.Printf("Looking at volume %q#%d", v.Name, i)
		switch {
		case v.EmptyDir != nil:
			dir := filepath.Join(varrun, id)
			dir = filepath.Join(dir, emptyDir)
			dir = filepath.Join(dir, fmt.Sprintf("#%d", i))
			if err := os.MkdirAll(dir, 2750); err != nil {
				return nil, err
			}
			if err := chown(dir, uid, gid); err != nil {
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

			dir := filepath.Join(varrun, id)
			dir = filepath.Join(dir, secretDir)
			dir = filepath.Join(dir, fmt.Sprintf("#%d", i))
			if err := os.MkdirAll(dir, 2750); err != nil {
				return nil, err
			}
			if err := chown(dir, uid, gid); err != nil {
				return nil, err
			}

			log.Printf("Created %q for secret: %s", dir, v.Name)

			for k, v := range secret.StringData {
				path := filepath.Join(dir, k)

				log.Printf("Writing secret to path %q", path)
				data, err := base64.StdEncoding.DecodeString(string(v))
				if err != nil {
					return vol, err
				}
				if err := ioutil.WriteFile(path, data, 0640); err != nil {
					return vol, err
				}
			}
			for k, v := range secret.Data {
				path := filepath.Join(dir, k)

				log.Printf("Writing secret to path %q", path)
				err := ioutil.WriteFile(path, []byte(v), 0640)
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

			dir := filepath.Join(varrun, id)
			dir = filepath.Join(dir, configmapDir)
			dir = filepath.Join(dir, fmt.Sprintf("#%d", i))
			if err := os.MkdirAll(dir, 2750); err != nil {
				return nil, err
			}
			if err := chown(dir, uid, gid); err != nil {
				return nil, err
			}

			log.Printf("Created %q for configmap: %s", dir, v.Name)

			for k, v := range configMap.Data {
				path := filepath.Join(dir, k)
				log.Printf("Writing configMap Data to path %q", path)
				if err := ioutil.WriteFile(path, []byte(v), 0640); err != nil {
					return vol, err
				}
			}
			for k, v := range configMap.BinaryData {
				path := filepath.Join(dir, k)
				log.Printf("Writing configMap BinaryData to path %q", path)
				err := ioutil.WriteFile(path, v, 0640)
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

// chown chowns name with uid and gid.
func chown(name, uid, gid string) error {
	// we're parsing uid/gid back and forth
	uidn, err := strconv.ParseInt(uid, 10, 64)
	if err != nil {
		uidn = -1
	}
	gidn, err := strconv.ParseInt(gid, 10, 64)
	if err != nil {
		gidn = -1
	}

	return os.Chown(name, int(uidn), int(gidn))
}

func isEmpty(name string) (bool, error) {
	d, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer d.Close()

	_, err = d.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
