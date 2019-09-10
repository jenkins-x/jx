package watcher

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// Watcher lists and watches resources
type Watcher struct {
	Namespace    string
	ListOptions  metav1.ListOptions
	HandlerFuncs HandlerFuncs
	ListWatch    ListWatch
}

// HandlerFuncs is an adaptor to let you easily specify as many or
// as few of the notification functions as you want
type HandlerFuncs struct {
	AddFunc    func(obj runtime.Object)
	ModifyFunc func(obj runtime.Object)
	DeleteFunc func(obj runtime.Object)
}

// ListWatch an interface for something that can list and watch resources
type ListWatch interface {
	List(ns string, listOptions metav1.ListOptions) ([]runtime.Object, error)
	Watch(ns string, listOptions metav1.ListOptions) (watch.Interface, error)
}

// Run watches the resources until the connection is lost
func (w *Watcher) Run() error {
	log.Logger().Infof("watching and listing resources in namespace %s", w.Namespace)

	watcher, err := w.ListWatch.Watch(w.Namespace, w.ListOptions)
	if err != nil {
		log.Logger().Fatalf("failed to watch resources in namespace %s due to %s", w.Namespace, err.Error())
	}

	resources, err := w.ListWatch.List(w.Namespace, w.ListOptions)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Logger().Fatalf("failed to list resources in namespace %s due to %s", w.Namespace, err.Error())
		}
	}

	for _, resource := range resources {
		w.onAdd(resource)
	}

	// TODO we could ensure that watch events are always newer than the first list just in case?
	ch := watcher.ResultChan()
	for event := range ch {
		switch event.Type {
		case watch.Added:
			w.onAdd(event.Object)
		case watch.Modified:
			w.onModified(event.Object)
		case watch.Deleted:
			w.onDeleted(event.Object)
		case watch.Error:
			log.Logger().Fatalf("watcher is closed in namespace %s", w.Namespace)
			watcher.Stop()
			return fmt.Errorf("watch channel disconnected")
		}
	}
	return nil
}

func (w *Watcher) onAdd(object runtime.Object) {
	if w.HandlerFuncs.AddFunc != nil {
		w.HandlerFuncs.AddFunc(object)
	}
}

func (w *Watcher) onModified(object runtime.Object) {
	if w.HandlerFuncs.ModifyFunc != nil {
		w.HandlerFuncs.ModifyFunc(object)
	}
}

func (w *Watcher) onDeleted(object runtime.Object) {
	if w.HandlerFuncs.DeleteFunc != nil {
		w.HandlerFuncs.DeleteFunc(object)
	}
}

func (w *Watcher) onError() {
	log.Logger().Fatalf("watcher is closed in namespace %s", w.Namespace)
}
