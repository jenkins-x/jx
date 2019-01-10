package kube

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/log"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"sort"
	"time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineNamespaceCache caches the pipelines for a single namespace
type PipelineNamespaceCache struct {
	pipelines map[string]*v1.PipelineActivity
}

// NewPipelineCache creates a cache of pipelines for a namespace
func NewPipelineCache(jxClient versioned.Interface, ns string) *PipelineNamespaceCache {
	pipeline := &v1.PipelineActivity{}
	stop := make(chan struct{})
	pipelineListWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "pipelineactivities", ns, fields.Everything())

	namespaceCache := &PipelineNamespaceCache{
		pipelines: map[string]*v1.PipelineActivity{},
	}

	// lets pre-populate the cache on startup as there's not yet a way to know when the informer has completed its first list operation
	pipelines, _ := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
	if pipelines != nil {
		for _, pipeline := range pipelines.Items {
			copy := pipeline
			namespaceCache.pipelines[pipeline.Name] = &copy
		}
	}
	_, pipelineController := cache.NewInformer(
		pipelineListWatch,
		pipeline,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				namespaceCache.onPipelineObj(obj, jxClient, ns)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				namespaceCache.onPipelineObj(newObj, jxClient, ns)
			},
			DeleteFunc: func(obj interface{}) {
				namespaceCache.onPipelineDelete(obj, jxClient, ns)
			},
		},
	)

	go pipelineController.Run(stop)

	return namespaceCache
}

// Pipelines returns the pipelines in this namespace sorted in name order
func (c *PipelineNamespaceCache) Pipelines() []*v1.PipelineActivity {
	answer := []*v1.PipelineActivity{}
	names := []string{}
	for k := range c.pipelines {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		pipeline := c.pipelines[name]
		if pipeline != nil {
			answer = append(answer, pipeline)
		}
	}
	return answer
}

func (c *PipelineNamespaceCache) onPipelineObj(obj interface{}, jxClient versioned.Interface, ns string) {
	pipeline, ok := obj.(*v1.PipelineActivity)
	if !ok {
		log.Warnf("Object is not a PipelineActivity %#v\n", obj)
		return
	}
	if pipeline != nil {
		c.pipelines[pipeline.Name] = pipeline
	}
}

func (c *PipelineNamespaceCache) onPipelineDelete(obj interface{}, jxClient versioned.Interface, ns string) {
	pipeline, ok := obj.(*v1.PipelineActivity)
	if !ok {
		log.Warnf("Object is not a PipelineActivity %#v\n", obj)
		return
	}
	if pipeline != nil {
		delete(c.pipelines, pipeline.Name)
	}
}
