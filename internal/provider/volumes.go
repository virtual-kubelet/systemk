package provider

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	emptyDir     = "emptydirs"
	secretDir    = "secrets"
	configmapDir = "configmaps"
)

// Volume describes what volumes should be created.
type Volume int

const (
	volumeAll Volume = iota
	volumeConfigMap
	volumeSecret
)

// volumes inspects the PodSpec.Volumes attribute and returns a mapping with the volume's Name and the directory on-disk that
// should be used for this. The on-disk structure is prepared and can be used.
// which considered what volumes should be setup. Defaults to volumeAll.
func (p *p) volumes(pod *corev1.Pod, which Volume) (map[string]string, error) {
	fnlog := log.
		WithField("podNamespace", pod.Namespace).
		WithField("podName", pod.Name)

	vol := make(map[string]string)
	id := string(pod.ObjectMeta.UID)
	uid, gid := uidGidFromSecurityContext(pod)
	for i, v := range pod.Spec.Volumes {
		fnlog.Debugf("looking at volume %q#%d", v.Name, i)
		switch {
		case v.HostPath != nil:
			if which != volumeAll {
				continue
			}
			// should this be v.Path? We should make this writeable
			// We should also check the allowed paths here. TODO(miek).
			vol[v.Name] = ""

		case v.EmptyDir != nil:
			if which != volumeAll {
				continue
			}
			dir := filepath.Join(p.runDir, id)
			dir = filepath.Join(dir, emptyDir)
			if err := isBelow(p.config.AllowedHostPaths, dir); err != nil {
				return nil, err
			}
			if err := mkdirAllChown(dir, dirPerms, uid, gid); err != nil {
				return nil, err
			}
			dir = filepath.Join(dir, fmt.Sprintf("#%d", i))
			if err := mkdirAllChown(dir, dirPerms, uid, gid); err != nil {
				return nil, err
			}
			fnlog.Debugf("created %q for emptyDir %q", dir, v.Name)
			vol[v.Name] = dir

		case v.Secret != nil:
			if which != volumeAll && which != volumeSecret {
				continue
			}
			secret, err := p.podResourceManager.SecretLister().Secrets(pod.Namespace).Get(v.Secret.SecretName)
			if v.Secret.Optional != nil && !*v.Secret.Optional && errors.IsNotFound(err) {
				return nil, fmt.Errorf("secret %s is required by pod %s and does not exist", v.Secret.SecretName, pod.Name)
			}
			if secret == nil {
				continue
			}

			dir := filepath.Join(p.runDir, id)
			dir = filepath.Join(dir, secretDir)
			if err := isBelow(p.config.AllowedHostPaths, dir); err != nil {
				return nil, err
			}
			if err := mkdirAllChown(dir, dirPerms, uid, gid); err != nil {
				return nil, err
			}
			dir = filepath.Join(dir, fmt.Sprintf("#%d", i))
			if err := mkdirAllChown(dir, dirPerms, uid, gid); err != nil {
				return nil, err
			}
			fnlog.Debugf("created %q for secret %q", dir, v.Name)

			for k, v := range secret.StringData {
				data, err := base64.StdEncoding.DecodeString(string(v))
				if err != nil {
					return nil, err
				}
				if err := writeFile(dir, k, uid, gid, data); err != nil {
					return nil, err
				}
			}
			for k, v := range secret.Data {
				if err := writeFile(dir, k, uid, gid, []byte(v)); err != nil {
					return nil, err
				}
			}
			vol[v.Name] = dir

		case v.ConfigMap != nil:
			if which != volumeAll && which != volumeConfigMap {
				continue
			}
			configMap, err := p.podResourceManager.ConfigMapLister().ConfigMaps(pod.Namespace).Get(v.ConfigMap.Name)
			if v.ConfigMap.Optional != nil && !*v.ConfigMap.Optional && errors.IsNotFound(err) {
				return nil, fmt.Errorf("configMap %s is required by pod %s and does not exist", v.ConfigMap.Name, pod.Name)
			}
			if configMap == nil {
				continue
			}

			dir := filepath.Join(p.runDir, id)
			dir = filepath.Join(dir, configmapDir)
			if err := isBelow(p.config.AllowedHostPaths, dir); err != nil {
				return nil, err
			}
			if err := mkdirAllChown(dir, dirPerms, uid, gid); err != nil {
				return nil, err
			}
			dir = filepath.Join(dir, fmt.Sprintf("#%d", i))
			if err := mkdirAllChown(dir, dirPerms, uid, gid); err != nil {
				return nil, err
			}
			fnlog.Debugf("created %q for configmap %q", dir, v.Name)

			for k, v := range configMap.Data {
				if err := writeFile(dir, k, uid, gid, []byte(v)); err != nil {
					return nil, err
				}
			}
			for k, v := range configMap.BinaryData {
				if err := writeFile(dir, k, uid, gid, v); err != nil {
					return nil, err
				}
			}
			vol[v.Name] = dir

		default:
			return nil, fmt.Errorf("pod %s requires volume %s which is of an unsupported type", pod.Name, v.Name)
		}
	}

	return vol, nil
}

func (p *p) cleanPodEphemeralVolumes(podId string) error {
	podEphemeralVolumes := filepath.Join(p.runDir, podId)
	return os.RemoveAll(podEphemeralVolumes)
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

// mkdirAllChown calls os.MkdirAll and chown to create path and set ownership.
func mkdirAllChown(path string, perm os.FileMode, uid, gid string) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}

	return chown(path, uid, gid)
}

// writeFile writes data in to a tmp file in dir and chowns it to uid/gid and
// then moves it over file. Note 'file': This mv fails for directories. Those need
// to be removed first? (This is not done yet)
func writeFile(dir, file, uid, gid string, data []byte) error {
	fnlog := log.
		WithField("dir", dir).
		WithField("file", file)

	tmpfile, err := ioutil.TempFile(dir, "systemk.*.tmp")
	if err != nil {
		return err
	}
	fnlog.Debugf("chowning %q to %s.%s", tmpfile.Name(), uid, gid)
	if err := chown(tmpfile.Name(), uid, gid); err != nil {
		return err
	}

	x := 10
	if len(data) < 10 {
		x = len(data)
	}
	fnlog.Debugf("writing data %q to path %q", data[:x], tmpfile.Name())
	if err := ioutil.WriteFile(tmpfile.Name(), data, 0640); err != nil {
		return err
	}
	path := filepath.Join(dir, file)
	fnlog.Debugf("renaming %q to %q", tmpfile.Name(), path)

	return os.Rename(tmpfile.Name(), path)
}

// chown chowns name with uid and gid.
func chown(name, uid, gid string) error {
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

// isBelowPath returns true if path is below top.
func isBelowPath(top, path string) bool {
	x, err := filepath.Rel(top, path)
	if err != nil {
		return false
	}
	if x == "." { // same level
		return false
	}
	return !strings.Contains(x, "..")
}

// isBelowPath checks if path falls below any of the path in dirs.
func isBelow(dirs []string, path string) error {
	for _, d := range dirs {
		if isBelowPath(d, path) {
			return nil
		}
	}
	return fmt.Errorf("path %q, doesn't fall under any paths in %s", path, dirs)
}

const dirPerms = 02750
