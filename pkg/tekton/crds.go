package tekton

import (
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

// CRDWrapper is a wrapper around the various Tekton CRDs
type CRDWrapper struct {
	Pipeline       *pipelineapi.Pipeline
	Tasks          []*pipelineapi.Task
	Resources      []*pipelineapi.PipelineResource
	PipelineRun    *pipelineapi.PipelineRun
	Structure      *v1.PipelineStructure
	PipelineParams []pipelineapi.Param
}

// ObjectReferences creates a list of object references created
func (r *CRDWrapper) ObjectReferences() []kube.ObjectReference {
	resources := []kube.ObjectReference{}
	for _, task := range r.Tasks {
		if task.ObjectMeta.Name == "" {
			log.Warnf("created Task has no name: %#v\n", task)

		} else {
			resources = append(resources, kube.CreateObjectReference(task.TypeMeta, task.ObjectMeta))
		}
	}
	if r.Pipeline != nil {
		if r.Pipeline.ObjectMeta.Name == "" {
			log.Warnf("created Pipeline has no name: %#v\n", r.Pipeline)

		} else {
			resources = append(resources, kube.CreateObjectReference(r.Pipeline.TypeMeta, r.Pipeline.ObjectMeta))
		}
	}
	if r.PipelineRun != nil {
		if r.PipelineRun.ObjectMeta.Name == "" {
			log.Warnf("created PipelineRun has no name: %#v\n", r.PipelineRun)
		} else {
			resources = append(resources, kube.CreateObjectReference(r.PipelineRun.TypeMeta, r.PipelineRun.ObjectMeta))
		}
	}
	if len(resources) == 0 {
		log.Warnf("no Tasks, Pipeline or PipelineRuns created\n")
	}
	return resources
}
