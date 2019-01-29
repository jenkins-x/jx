// Package buildnum contains stuff to do with generating build numbers.
package buildnum

import (
	"sync"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"

	v1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
)

// PipelineActivityBuildNumGen generates build numbers backed by PipelineActivity K8S CRDs.
type PipelineActivityBuildNumGen struct {
	//Protect access to pipelineMutexes map.
	mutex *sync.Mutex
	//pipelineID->Mutex to serialise build number generation for a given pipeline ID.
	pipelineMutexes  map[string]*sync.Mutex
	activitiesGetter v1.PipelineActivityInterface
	pipelineCache    *kube.PipelineNamespaceCache
}

// NewCRDBuildNumGen initialises a new PipelineActivityBuildNumGen that will use the supplied
// PipelineActivityInterface to query CRDs.
func NewCRDBuildNumGen(jxClient versioned.Interface, ns string) *PipelineActivityBuildNumGen {
	return &PipelineActivityBuildNumGen{
		mutex:            &sync.Mutex{},
		pipelineMutexes:  make(map[string]*sync.Mutex),
		pipelineCache:    kube.NewPipelineCache(jxClient, ns),
		activitiesGetter: jxClient.JenkinsV1().PipelineActivities(ns),
	}
}

// Ready returns true if the generator's cache has done its initial load.
func (g *PipelineActivityBuildNumGen) Ready() bool {
	return g.pipelineCache.Ready()
}

// NextBuildNumber returns the next build number for the specified pipeline ID, storing the sequence in K8S.
// Returns the build number, or an error if there is a problem with K8S resources.
func (g *PipelineActivityBuildNumGen) NextBuildNumber(pipeline kube.PipelineID) (string, error) {
	g.mutex.Lock()

	//Find a mutex for this pipelineId.
	pipelineMutex, ok := g.pipelineMutexes[pipeline.ID]
	if !ok {
		pipelineMutex = &sync.Mutex{}
		g.pipelineMutexes[pipeline.ID] = pipelineMutex
	}
	pipelineMutex.Lock()
	g.mutex.Unlock()

	defer func() {
		g.mutex.Lock()
		pipelineMutex.Unlock()
		delete(g.pipelineMutexes, pipeline.ID)
		g.mutex.Unlock()
	}()

	pipelines := g.pipelineCache.Pipelines()
	buildNum, _, err := kube.GenerateBuildNumber(g.activitiesGetter, pipelines, pipeline)

	if err != nil {
		return "", err
	}
	return buildNum, nil
}
