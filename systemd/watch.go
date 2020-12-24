package systemd

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// The Updater interface we will send notifications when configmaps and secrets are updated.
type Updater interface {
	updateConfigMap(context.Context, *corev1.Pod, *corev1.ConfigMap) error
	updateSecret(context.Context, *corev1.Pod, *corev1.Secret) error
}

// Watcher checks the API server for configMap and secret updates and notifies the provider.
type Watcher struct {
	mu sync.RWMutex
	// maps are indexed by "namespace.name"
	configs map[string][]*corev1.Pod
	secrets map[string][]*corev1.Pod
}

func newWatcher() *Watcher {
	return &Watcher{
		configs: make(map[string][]*corev1.Pod),
		secrets: make(map[string][]*corev1.Pod),
	}
}

func (w *Watcher) handlerFuncs(p Updater) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.handleEvent(obj, p)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.handleEvent(newObj, p)
		},
		DeleteFunc: nil, // Ignore for now. Pod deletion takes care of cleaning up.
	}
}
func (w *Watcher) handleEvent(obj interface{}, upd Updater) {
	switch v := obj.(type) {
	case *corev1.Secret:
		w.mu.RLock()
		pods := w.secrets[v.Namespace+"."+v.Name]
		w.mu.RUnlock()
		if len(pods) != 0 {
			klog.Infof("Secret update %s/%s, notifying %d pods", v.Namespace, v.Name, len(pods))
		}
		for _, pod := range pods {
			if err := upd.updateSecret(context.TODO(), pod, v); err != nil {
				klog.Warningf("Secret update %s/%s: %s", v.Namespace, v.Name, err)
			}
		}
	case *corev1.ConfigMap:
		w.mu.RLock()
		pods := w.configs[v.Namespace+"."+v.Name]
		w.mu.RUnlock()
		if len(pods) != 0 {
			klog.Infof("ConfigMap update %s/%s, notifying %d pods", v.Namespace, v.Name, len(pods))
		}
		for _, pod := range pods {
			if err := upd.updateConfigMap(context.TODO(), pod, v); err != nil {
				klog.Warningf("ConfigMap update %s/%s: %s", v.Namespace, v.Name, err)
			}
		}
	default:
		klog.Warningf("Ignoring unsupported type %T", v)
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
