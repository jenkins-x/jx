package build_num

import (
	"sync"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
)

//A type for generating build numbers backed by PipelineActivity K8S CRDs.
type PipelineActivityBuildNumGen struct {
	//Protect access to pipelineMutexes map.
	mutex *sync.Mutex
	//pipelineID->Mutex to serialise build number generation for a given pipeline ID.
	pipelineMutexes  map[string]*sync.Mutex
	activitiesGetter v1.PipelineActivityInterface
}

func NewCRDBuildNumGen(activitiesGetter v1.PipelineActivityInterface) *PipelineActivityBuildNumGen {
	return &PipelineActivityBuildNumGen{
		mutex:            &sync.Mutex{},
		pipelineMutexes:  make(map[string]*sync.Mutex),
		activitiesGetter: activitiesGetter,
	}
}

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

	buildNum, _, err := kube.GenerateBuildNumber(g.activitiesGetter, pipeline)

	if err != nil {
		return "", err
	}
	return buildNum, nil
}
