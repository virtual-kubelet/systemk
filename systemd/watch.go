package systemd

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// The updater notifies pods when related configmaps and secrets change.
type updater interface {
	updateConfigMap(ctx context.Context, pod *corev1.Pod, configMap *corev1.ConfigMap) error
	updateSecret(ctx context.Context, pod *corev1.Pod, secret *corev1.Secret) error
}

// watcher checks the API server for configMap and secret updates and notifies the provider.
type watcher struct {
	mu sync.RWMutex
	// must keep a deep-copy of Pod since its volumes are gone when deletion is triggered.
	configs map[types.NamespacedName][]*corev1.Pod
	// cmKeysByPod enables reverse lookup ConfigMaps keys per Pod.
	cmKeysByPod map[types.NamespacedName][]types.NamespacedName

	// must keep a deep-copy of Pod since its volumes are gone when deletion is triggered.
	secrets map[types.NamespacedName][]*corev1.Pod
	// secretKeysByPod enables reverse lookup Secrets keys per Pod.
	secretKeysByPod map[types.NamespacedName][]types.NamespacedName
}

func newWatcher() *watcher {
	return &watcher{
		configs:         make(map[types.NamespacedName][]*corev1.Pod),
		cmKeysByPod:     make(map[types.NamespacedName][]types.NamespacedName),
		secrets:         make(map[types.NamespacedName][]*corev1.Pod),
		secretKeysByPod: make(map[types.NamespacedName][]types.NamespacedName),
	}
}

func (w *watcher) handlerFuncs(ctx context.Context, p updater) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.handleEvent(ctx, obj, p)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.handleEvent(ctx, newObj, p)
		},
		DeleteFunc: nil, // Ignore for now. Pod deletion takes care of cleaning up.
	}
}
func (w *watcher) handleEvent(ctx context.Context, obj interface{}, upd updater) {
	switch v := obj.(type) {
	case *corev1.ConfigMap:
		w.mu.RLock()
		cmKey := types.NamespacedName{Namespace: v.Namespace, Name: v.Name}
		pods := w.configs[cmKey]
		w.mu.RUnlock()
		if len(pods) != 0 {
			klog.Infof("Secret update %s/%s, notifying %d pods", v.Namespace, v.Name, len(pods))
		}
		for _, pod := range pods {
			if err := upd.updateConfigMap(ctx, pod, v); err != nil {
				klog.Warningf("Secret update %s/%s: %s", v.Namespace, v.Name, err)
			}
		}

	case *corev1.Secret:
		w.mu.RLock()
		secretKey := types.NamespacedName{Namespace: v.Namespace, Name: v.Name}
		pods := w.secrets[secretKey]
		w.mu.RUnlock()
		if len(pods) != 0 {
			klog.Infof("ConfigMap update %s/%s, notifying %d pods", v.Namespace, v.Name, len(pods))
		}
		for _, pod := range pods {
			if err := upd.updateSecret(ctx, pod, v); err != nil {
				klog.Warningf("ConfigMap update %s/%s: %s", v.Namespace, v.Name, err)
			}
		}
	default:
		klog.Warningf("Ignoring unsupported type %T", v)
	}
}

// Watch observes the configmaps and secrets referenced by a pod.
func (w *watcher) Watch(pod *corev1.Pod) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	podKey := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}

	for _, v := range pod.Spec.Volumes {
		switch {
		case v.ConfigMap != nil:
			cmKey := types.NamespacedName{Namespace: pod.Namespace, Name: v.ConfigMap.Name}
			w.configs[cmKey] = append(w.configs[cmKey], pod.DeepCopy())
			w.cmKeysByPod[podKey] = append(w.cmKeysByPod[podKey], cmKey)

		case v.Secret != nil:
			secretKey := types.NamespacedName{Namespace: pod.Namespace, Name: v.Secret.SecretName}
			w.secrets[secretKey] = append(w.secrets[secretKey], pod.DeepCopy())
			w.secretKeysByPod[podKey] = append(w.secretKeysByPod[podKey], secretKey)
		}
	}
}

// Unwatch removes the watches for the pod.
func (w *watcher) Unwatch(pod *corev1.Pod) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	podKey := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}

	// reverse lookup all ConfigMap keys referenced by this Pod.
	cmKeys := w.cmKeysByPod[podKey]
	// Now, iterate over all ConfigMaps referenced by this Pod and remove it.
	for _, cmKey := range cmKeys {
		var watchedPods []*corev1.Pod
		for _, pod := range w.configs[cmKey] {
			if pod.Namespace == pod.Namespace && pod.Name == pod.Name {
				continue
			}
			watchedPods = append(watchedPods, pod)
		}

		if len(watchedPods) == 0 {
			delete(w.configs, cmKey)
			continue
		}
		w.configs[cmKey] = watchedPods
	}
	// No longer reverse lookup ConfigMaps for this pod.
	delete(w.cmKeysByPod, podKey)

	// reverse lookup all Secret keys referenced by this Pod.
	secretKeys := w.secretKeysByPod[podKey]
	// Now, iterate over all Secrets referenced by this Pod and remove it.
	for _, secretKey := range secretKeys {
		var watchedPods []*corev1.Pod
		for _, pod := range w.secrets[secretKey] {
			if pod.Namespace == pod.Namespace && pod.Name == pod.Name {
				continue
			}
			watchedPods = append(watchedPods, pod)
		}

		if len(watchedPods) == 0 {
			delete(w.secrets, secretKey)
			continue
		}
		w.secrets[secretKey] = watchedPods
	}
	// No longer reverse lookup Secrets for this pod.
	delete(w.secretKeysByPod, podKey)
}
