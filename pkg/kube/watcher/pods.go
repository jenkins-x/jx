package watcher

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type podListWatch struct {
	client kubernetes.Interface
}

// NewPodWatcher creates a new pod watcher
func NewPodWatcher(client kubernetes.Interface, ns string, listOptions metav1.ListOptions, handlerFuncs HandlerFuncs) Watcher {
	listWatch := NewPodListWatch(client)
	return Watcher{
		Namespace:    ns,
		ListOptions:  listOptions,
		HandlerFuncs: handlerFuncs,
		ListWatch:    listWatch,
	}
}

// NewPodListWatch creates a new pod list watch
func NewPodListWatch(client kubernetes.Interface) ListWatch {
	return &podListWatch{client: client}
}

// List lists the resources in the namespace
func (w *podListWatch) List(ns string, listOptions metav1.ListOptions) ([]runtime.Object, error) {
	resourceList, err := w.client.CoreV1().Pods(ns).List(listOptions)
	var answer []runtime.Object
	if resourceList != nil {
		for _, resource := range resourceList.Items {
			copy := resource
			answer = append(answer, &copy)
		}
	}
	return answer, err
}

// Watch creates a watcher of the resources in the namespace
func (w *podListWatch) Watch(ns string, listOptions metav1.ListOptions) (watch.Interface, error) {
	resources := w.client.CoreV1().Pods(ns)
	return resources.Watch(listOptions)
}
