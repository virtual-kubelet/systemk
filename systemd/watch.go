package systemd

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// The Updater interface we will send notifications when configmaps and secrets are updated.
type Updater interface {
	UpdateConfigMap(context.Context, *corev1.Pod, *corev1.ConfigMap) error
	UpdateSecret(context.Context, *corev1.Pod, *corev1.Secret) error
}

// Watcher checks the API server for configMap and secret updates and notifies the provider.
type Watcher struct {
	mu        sync.RWMutex
	clientset *kubernetes.Clientset
	// maps are indexed by "namespace.name"
	configs map[string][]*corev1.Pod
	secrets map[string][]*corev1.Pod
}

func newWatcher(clientset *kubernetes.Clientset) *Watcher {
	return &Watcher{
		configs:   make(map[string][]*corev1.Pod),
		secrets:   make(map[string][]*corev1.Pod),
		clientset: clientset,
	}
}

func (w *Watcher) run(p Updater) error {
	cmwatcher, err := w.clientset.CoreV1().ConfigMaps(corev1.NamespaceAll).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	swatcher, err := w.clientset.CoreV1().Secrets(corev1.NamespaceAll).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for {
		select {
		case event, ok := <-cmwatcher.ResultChan():
			if !ok {
				cmwatcher, _ = w.clientset.CoreV1().ConfigMaps(corev1.NamespaceAll).Watch(context.TODO(), metav1.ListOptions{})
				break
			}
			cm, ok := event.Object.(*corev1.ConfigMap)
			if !ok {
				continue
			}
			// event.Type == watch.Deleted ??
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}

			namespace := cm.ObjectMeta.Namespace
			name := cm.ObjectMeta.Name
			w.mu.RLock()
			pods := w.configs[namespace+"."+name]
			w.mu.RUnlock()
			if len(pods) != 0 {
				klog.Infof("ConfigMap update %s/%s, notifying %d pods", namespace, name, len(pods))
			}
			for _, pod := range pods {
				if err := p.UpdateConfigMap(context.TODO(), pod, cm); err != nil {
					klog.Warningf("ConfigMap update %s/%s: %s", namespace, name, err)
				}
			}
		case event, ok := <-swatcher.ResultChan():
			if !ok {
				swatcher, _ = w.clientset.CoreV1().Secrets(corev1.NamespaceAll).Watch(context.TODO(), metav1.ListOptions{})
				break
			}
			s, ok := event.Object.(*corev1.Secret)
			if !ok {
				continue
			}
			// event.Type == watch.Deleted ??
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}

			namespace := s.ObjectMeta.Namespace
			name := s.ObjectMeta.Name
			w.mu.RLock()
			pods := w.secrets[namespace+"."+name]
			w.mu.RUnlock()
			if len(pods) != 0 {
				klog.Infof("Secret update %s/%s, notifying %d pods", namespace, name, len(pods))
			}
			for _, pod := range pods {
				if err := p.UpdateSecret(context.TODO(), pod, s); err != nil {
					klog.Warningf("Secret update %s/%s: %s", namespace, name, err)
				}
			}
		}
	}
}

// Watch observes the config maps and secrets for the pod. It calls back into the
// provider using the Updater interface methods.
func (w *Watcher) Watch(pod *corev1.Pod) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, v := range pod.Spec.Volumes {
		switch {
		case v.ConfigMap != nil:
			key := pod.Namespace + "." + v.ConfigMap.Name
			w.configs[key] = append(w.configs[key], pod)

		case v.Secret != nil:
			key := pod.Namespace + "." + v.Secret.SecretName
			w.secrets[key] = append(w.secrets[key], pod)
		}
	}
}

// Unwatch removes the watches for the pod.
func (w *Watcher) Unwatch(pod *corev1.Pod) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, v := range pod.Spec.Volumes {
		switch {
		case v.ConfigMap != nil:
			key := pod.Namespace + "." + v.ConfigMap.Name
			// build new list; yes can be done more efficient.
			pods := []*corev1.Pod{}
			for _, p := range w.configs[key] {
				if p.Namespace == pod.Namespace && p.Name == pod.Name {
					continue
				}
				pods = append(pods, p)
			}
			if len(pods) == 0 {
				delete(w.configs, key)
				continue
			}
			w.configs[key] = pods

		case v.Secret != nil:
			key := pod.Namespace + "." + v.Secret.SecretName
			pods := []*corev1.Pod{}
			for _, p := range w.secrets[key] {
				if p.Namespace == pod.Namespace && p.Name == pod.Name {
					continue
				}
				pods = append(pods, p)
			}
			if len(pods) == 0 {
				delete(w.secrets, key)
				continue
			}
			w.secrets[key] = pods
		}
	}
}
