package watcher

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
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

type WatchChannel struct {
	handlerFuncs HandlerFuncs
	watcher      watch.Interface
	name         string
}

// CreateChannel lists resources and creates a watch channel for watching events
func (w *Watcher) CreateChannel(name string) (*WatchChannel, error) {
	log.Logger().Infof("watching and listing resources: %s in namespace %s", name, w.Namespace)

	watcher, err := w.ListWatch.Watch(w.Namespace, w.ListOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to watch resources in namespace %s", w.Namespace)
	}

	resources, err := w.ListWatch.List(w.Namespace, w.ListOptions)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "failed to list resources in namespace %s due", w.Namespace)
		}
	}
	for _, resource := range resources {
		w.HandlerFuncs.OnAdd(resource)
	}
	return &WatchChannel{
		handlerFuncs: w.HandlerFuncs,
		watcher:      watcher,
		name:         name,
	}, nil
}

// Stop stop the channel
func (c *WatchChannel) Stop() {
	c.watcher.Stop()
}

// Run watches the resources on the channel until the connection closes or a watch error is generated
func (c *WatchChannel) Run() error {
	// TODO we could ensure that watch events are always newer than the first list just in case?
	ch := c.watcher.ResultChan()
	w := c.handlerFuncs
	for event := range ch {
		switch event.Type {
		case watch.Added:
			w.OnAdd(event.Object)
		case watch.Modified:
			w.OnModified(event.Object)
		case watch.Deleted:
			w.OnDeleted(event.Object)
		case watch.Error:
			log.Logger().Errorf("watcher %s got error", c.name)
			c.watcher.Stop()
			return fmt.Errorf("WatchChannel error occurred")
		}
	}
	return nil
}

func (h *HandlerFuncs) OnAdd(object runtime.Object) {
	if h.AddFunc != nil {
		h.AddFunc(object)
	}
}

func (h *HandlerFuncs) OnModified(object runtime.Object) {
	if h.ModifyFunc != nil {
		h.ModifyFunc(object)
	}
}

func (h *HandlerFuncs) OnDeleted(object runtime.Object) {
	if h.DeleteFunc != nil {
		h.DeleteFunc(object)
	}
}
