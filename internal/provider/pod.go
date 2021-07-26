package provider

import (
	"context"
	"fmt"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/pkg/errors"
	"github.com/virtual-kubelet/systemk/internal/ospkg"
	"github.com/virtual-kubelet/systemk/internal/unit"
	nodeapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
)

// If any of these methods return an error, it will show up in the kubectl output as "ProviderFailed", so we should
// be very careful to just return one of something trivial failed. It's better to setup as much as you can then let
// the container/unit start fail, which will be correctly picked up by the control plane.

func (p *p) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	fnlog := log.WithField("podNamespace", namespace).WithField("podName", name)
	fnlog.Debug("GetPod called")
	unitprefix := unitPrefix(namespace, name) + separator // we need to closing dot here, otherwise will return update2, update3, when looking for update.
	stats, err := p.unitManager.States(unitprefix)
	if err != nil {
		fnlog.Errorf("failed to retrieve systemd states respective to Pod: %s", err)
		return nil, nil
	}
	pod := p.statsToPod(stats)
	return pod, nil
}

func (p *p) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	log.Debug("GetPods called")
	states, err := p.unitManager.States(prefix)
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, nil
	}

	// Get all the names and then we just call GetPod for each.
	ns := map[string][]string{} // namespace/ pod(s) mapping

	// sort unit by namespace/name
	for name := range states {
		namespace := Namespace(name)
		pod := Pod(name)
		ns[namespace] = append(ns[namespace], pod)
	}

	var pods []*corev1.Pod
	for namespace, names := range ns {
		for _, name := range names {
			if pod, err := p.GetPod(ctx, namespace, name); err != nil {
				pods = append(pods, pod)
			}
		}
	}
	return pods, nil
}

func (p *p) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	fnlog := log.
		WithField("podNamespace", pod.Namespace).
		WithField("podName", pod.Name)

	fnlog.Info("CreatePod called")

	vol, err := p.volumes(pod, volumeAll)
	if err != nil {
		err = errors.Wrap(err, "failed to process Pod volumes")
		fnlog.Error(err)
		return err
	}

	uid, gid, err := uidGidFromSecurityContext(pod, p.config.OverrideRootUID)
	if err != nil {
		return err
	}

	tmpfs := strings.Join([]string{"/var", "/run"}, " ")

	unitsToStart := []string{}
	previousUnit := ""
	for i, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		isInit := i < len(pod.Spec.InitContainers)
		fnlog.Debugf("processing container %d (init=%t)", i, isInit)

		// TODO(miek) parse c.Image for tag to get version. Check ImagePullAlways to reinstall??
		// if we're downloading the image, the image name needs cleaning
		installed, err := p.pkgManager.Install(c.Image, "")
		if err != nil {
			err = errors.Wrapf(err, "failed to install package %q", c.Image)
			fnlog.Error(err)
			return err
		}

		bindmounts := []string{}
		bindmountsro := []string{}
		rwpaths := []string{}
		for _, v := range c.VolumeMounts {
			dir, ok := vol[v.Name]
			if !ok {
				fnlog.Warnf("failed to find volumeMount %s in the specific volumes, skipping", v.Name)
				continue
			}

			if v.ReadOnly {
				bindmountsro = append(bindmountsro, fmt.Sprintf("%s:%s", dir, v.MountPath)) // SubPath, look at todo, filepath.Join?
				continue
			}
			rwpaths = append(rwpaths, v.MountPath)
			if dir == "" { // hostPath
				continue
			}
			bindmounts = append(bindmounts, fmt.Sprintf("%s:%s", dir, v.MountPath)) // SubPath, look at todo, filepath.Join?
			// OK, so the v.MountPath will _exist_ on the system, as systemd will create it, permissions should not matter, as we
			// only need this "hook" to mount the bindmount.
		}

		c.Image = ospkg.Clean(c.Image) // clean up the image if fetched with http(s)
		name := podToUnitName(pod, c.Name)
		if installed {
			p.unitManager.Mask(c.Image + unit.ServiceSuffix)
		}

		uf, err := p.unitfileFromPackageOrSynthesized(c)
		if err != nil {
			err = errors.Wrapf(err, "failed to process unit file for %q", c.Image)
			fnlog.Error(err)
			return err
		}
		if c.WorkingDir != "" {
			uf = uf.Overwrite("Service", "WorkingDirectory", c.WorkingDir)
		}

		uf = uf.Overwrite("Service", "ProtectSystem", "true")
		uf = uf.Overwrite("Service", "ProtectHome", "tmpfs")
		uf = uf.Overwrite("Service", "PrivateMounts", "true")
		uf = uf.Overwrite("Service", "ReadOnlyPaths", "/")
		uf = uf.Insert("Service", "StandardOutput", "journal")
		uf = uf.Insert("Service", "StandardError", "journal")

		// User/group handling. If the podspec has a security context we use that. This takes into acount the --override-root-uid flag value.
		// If these are not set, the unit file's value are used. Note if the unit file doesn't specify it, it *defaults*
		// to root, but we only care about that when a root override is set.
		hasRoot := false
		unitUser := uf.Contents["Service"]["User"]
		if len(unitUser) == 0 || unitUser[0] == "0" || unitUser[0] == "root" {
			hasRoot = true
		}
		unitGroup := uf.Contents["Service"]["Group"] // User=1, and a Group=0|root is also considered unwanted
		if len(unitGroup) == 0 || unitGroup[0] == "0" || unitGroup[0] == "root" {
			hasRoot = true
		}
		if uid == "" && hasRoot && p.config.OverrideRootUID > 0 {
			mapuid := strconv.FormatInt(int64(p.config.OverrideRootUID), 10)
			u, err := user.LookupId(mapuid)
			if err != nil {
				return fmt.Errorf("root override UID %q, not found: %s", mapuid, err)
			}
			uid = u.Uid
			gid = u.Gid
		}

		if uid != "" {
			uf = uf.Overwrite("Service", "User", uid)
			uf = uf.Overwrite("Service", "Group", gid)
		}

		// Treat initContainer differently.
		if isInit {
			uf = uf.Overwrite("Service", "Type", "oneshot") // no restarting
			uf = uf.Insert(kubernetesSection, "InitContainer", "true")
		}

		// Handle unit dependencies.
		if previousUnit != "" {
			uf = uf.Insert("Unit", "After", previousUnit)
		}

		// keep the unit around, until DeletePod is triggered.
		// this is also for us to return the state even after the unit left the stage.
		uf = uf.Overwrite("Service", "RemainAfterExit", "true")

		execStart := commandAndArgs(uf, c)
		if len(execStart) > 0 {
			uf = uf.Overwrite("Service", "ExecStart", strings.Join(execStart, " "))
		}

		id := string(pod.ObjectMeta.UID) // give multiple containers the same access? Need to test this.
		uf = uf.Insert(kubernetesSection, "Namespace", pod.ObjectMeta.Namespace)
		uf = uf.Insert(kubernetesSection, "ClusterName", pod.ObjectMeta.ClusterName)
		uf = uf.Insert(kubernetesSection, "Id", id)
		uf = uf.Insert(kubernetesSection, "Image", c.Image) // save (cleaned) image name here, we're not tracking this in the unit's name.

		uf = uf.Insert("Service", "TemporaryFileSystem", tmpfs)
		if len(rwpaths) > 0 {
			paths := strings.Join(rwpaths, " ")
			uf = uf.Insert("Service", "ReadWritePaths", paths)
		}
		if len(bindmounts) > 0 {
			mount := strings.Join(bindmounts, " ")
			uf = uf.Insert("Service", "BindPaths", mount)
		}
		if len(bindmountsro) > 0 {
			romount := strings.Join(bindmountsro, " ")
			uf = uf.Insert("Service", "BindReadOnlyPaths", romount)
		}

		for _, del := range deleteOptions {
			uf = uf.Delete("Service", del)
		}

		envVars := p.defaultEnvironment()
		for _, env := range c.Env {
			// If environment variable is a string with spaces, it must be quoted.
			// Quoting seems innocuous to other strings so it's set by default.
			envVars = append(envVars, fmt.Sprintf("%s=%q", env.Name, env.Value))
		}
		for _, env := range envVars {
			uf = uf.Insert("Service", "Environment", env)
		}

		// For logging purposes only.
		init := ""
		if isInit {
			init = "init-"
		}
		fnlog.Infof("loading %sunit %q, %q as %q\n%s", init, c.Name, c.Image, name, uf)
		if err := p.unitManager.Load(name, *uf); err != nil {
			fnlog.Errorf("failed to load unit %q: %s", name, err)
		}
		unitsToStart = append(unitsToStart, name)
		if isInit {
			previousUnit = name
		}
	}
	for _, name := range unitsToStart {
		fnlog.Infof("starting unit %q", name)
		if err := p.unitManager.TriggerStart(name); err != nil {
			fnlog.Errorf("failed to trigger start for unit %q: %s", name, err)
		}
	}
	p.podResourceManager.Watch(pod)
	return nil
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *p) RunInContainer(ctx context.Context, namespace, name, container string, cmd []string, attach nodeapi.AttachIO) error {
	log.
		WithField("podNamespace", namespace).
		WithField("podName", name).
		WithField("containerName", container).
		Debug("RunInContainer called")

	// Should we just try to start something? But with what user???
	return nil
}

// GetPodStatus returns the status of a pod by name that is running.
// returns nil if a pod by that name is not found.
func (p *p) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	log.
		WithField("podNamespace", namespace).
		WithField("podName", name).
		Debug("GetPodStatus called")
	pod, err := p.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if pod == nil {
		return nil, nil
	}

	return &pod.Status, nil
}

// JournalReader returns the actual journal reader.
// This is useful when an io.ReadCloser is not enough, eg we need Follow().
func JournalReader(namespace, name, container string, logOpts nodeapi.ContainerLogOpts) (*sdjournal.JournalReader, error) {
	// TODO(pires) show logs from the current Pod alone https://github.com/virtual-kubelet/systemk/issues/5#issuecomment-765278538
	fnlog := log.
		WithField("podNamespace", namespace).
		WithField("podName", name).
		WithField("containerName", container)

	fnlog.Infof("calling for container logs with options %+v", logOpts)

	unitName := strings.Join([]string{unitPrefix(namespace, name), container, "service"}, separator)
	journalConfig := sdjournal.JournalReaderConfig{
		Matches: []sdjournal.Match{
			{
				// Filter by unit.
				Field: sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT,
				Value: unitName,
			},
		},
	}
	if logOpts.SinceSeconds > 0 {
		// Since duration must be negative so we get logs from the past.
		journalConfig.Since = -time.Second * time.Duration(logOpts.SinceSeconds)
	}
	// By default, SinceTime is "0001-01-01 00:00:00 +0000 UTC".
	if !logOpts.SinceTime.IsZero() {
		journalConfig.Since = time.Since(logOpts.SinceTime)
	}
	if logOpts.Tail > 0 {
		journalConfig.NumFromTail = uint64(logOpts.Tail)
	}
	// By default, timestamps are present in journal entries.
	// Kubernetes defaults to not having timestamps, so we adapt.
	if !logOpts.Timestamps {
		journalConfig.Formatter = func(entry *sdjournal.JournalEntry) (string, error) {
			msg, ok := entry.Fields[sdjournal.SD_JOURNAL_FIELD_MESSAGE]
			if !ok {
				return "", fmt.Errorf("no %q field present in journal entry", sdjournal.SD_JOURNAL_FIELD_MESSAGE)
			}

			return fmt.Sprintf("%s\n", msg), nil
		}
	}

	journalReader, err := sdjournal.NewJournalReader(journalConfig)
	if err != nil {
		fnlog.Error("failed to retrieve logs from journald, for unit %q", unitName, err)
	}

	return journalReader, err
}

// UpdatePod is a noop,
func (p *p) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	log.
		WithField("podNamespace", pod.Namespace).
		WithField("podName", pod.Name).
		Debug("UpdatePod called (no-op)")

	return nil
}

// DeletePod deletes a pod.
func (p *p) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	fnlog := log.
		WithField("podNamespace", pod.Namespace).
		WithField("podName", pod.Name)

	fnlog.Info("DeletePod called")

	unitsToUnload := []string{}
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		name := podToUnitName(pod, c.Name)
		if err := p.unitManager.TriggerStop(name); err != nil {
			fnlog.Warnf("failed to trigger stop for unit %q: %s", name, err)
		}
		unitsToUnload = append(unitsToUnload, name)
	}

	for _, name := range unitsToUnload {
		if err := p.unitManager.Unload(name); err != nil {
			fnlog.Warnf("failed to unload unit %q: %s", name, err)
		}
		fnlog.Infof("deleted unit %q successfully", name)
	}
	p.unitManager.Reload()
	p.podResourceManager.Unwatch(pod)

	// Clean-up volumes.
	if err := cleanPodEphemeralVolumes(string(pod.UID)); err != nil {
		fnlog.Warn("failed to clean-up volumes: %s", err)
	}

	return nil
}

func (p *p) UpdateConfigMap(ctx context.Context, pod *corev1.Pod, cm *corev1.ConfigMap) error {
	_, err := p.volumes(pod, volumeConfigMap)
	return err
}

func (p *p) UpdateSecret(ctx context.Context, pod *corev1.Pod, s *corev1.Secret) error {
	_, err := p.volumes(pod, volumeSecret)
	return err
}

func podToUnitName(pod *corev1.Pod, containerName string) string {
	return unitPrefix(pod.Namespace, pod.Name) + separator + containerName + unit.ServiceSuffix
}

func unitPrefix(namespace, podName string) string {
	return prefix + separator + namespace + separator + podName
}

// Name returns <namespace>.<podname> from a well formed name.
// Units are named as 'systemk.<namespace>.<podname>.<container>'.
func Name(name string) string {
	el := strings.Split(name, separator)
	if len(el) < 4 {
		return ""
	}
	return el[1] + separator + el[2]
}

// Container returns the <container> from the well formed name. See Name.
func Container(name string) string {
	el := strings.Split(name, separator)
	if len(el) < 4 {
		return ""
	}
	return el[3]
}

// Pod returns <podname> from the well formed name. See Name.
func Pod(name string) string {
	el := strings.Split(name, separator)
	if len(el) < 4 {
		return ""
	}
	return el[2]
}

// Namespace returns <namespace> from the well formed name. See Name.
func Namespace(name string) string {
	el := strings.Split(name, separator)
	if len(el) < 4 {
		return ""
	}
	return el[1]
}

const (
	// prefix the unit file prefix we used.
	prefix    = "systemk"
	separator = "."
)

// deleteOptions has a list of options we will always delete from the unit files
// as they clash with the podSpec.
var deleteOptions = []string{"EnvironmentFile"}
