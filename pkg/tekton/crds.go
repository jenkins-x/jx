package tekton

import (
	"context"
	"github.com/ghodss/yaml"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"io/ioutil"
	"os"
	"path/filepath"
)

// CRDWrapper is a wrapper around the various Tekton CRDs
type CRDWrapper struct {
	pipeline    *pipelineapi.Pipeline
	tasks       []*pipelineapi.Task
	resources   []*pipelineapi.PipelineResource
	pipelineRun *pipelineapi.PipelineRun

	structure *v1.PipelineStructure
}

// NewCRDWrapper creates a new wrapper for all required Tekton CRDs.
func NewCRDWrapper(pipeline *pipelineapi.Pipeline,
	tasks []*pipelineapi.Task,
	resources []*pipelineapi.PipelineResource,
	structure *v1.PipelineStructure,
	run *pipelineapi.PipelineRun) (*CRDWrapper, error) {

	crds := &CRDWrapper{
		pipeline:    pipeline,
		tasks:       tasks,
		resources:   resources,
		structure:   structure,
		pipelineRun: run,
	}

	err := crds.validate()
	if err != nil {
		return nil, err
	}

	return crds, nil
}

// Name returns the name of the Pipeline.
func (crds *CRDWrapper) Name() string {
	return crds.pipeline.Name
}

// TODO Should probably return values not pointers in these methods, but requires changes of all test data as well (HF)

// Pipeline returns a pointer to the Tekton Pipeline.
func (crds *CRDWrapper) Pipeline() *pipelineapi.Pipeline {
	return crds.pipeline
}

// Tasks returns an array of pointers to Tekton Tasks.
func (crds *CRDWrapper) Tasks() []*pipelineapi.Task {
	return crds.tasks
}

// PipelineRun returns a pointers to Tekton PipelineRun.
func (crds *CRDWrapper) PipelineRun() *pipelineapi.PipelineRun {
	return crds.pipelineRun
}

// Structure returns a pointers to Tekton PipelineStructure.
func (crds *CRDWrapper) Structure() *v1.PipelineStructure {
	return crds.structure
}

// Resources returns an array of pointers to Tekton PipelineResource.
func (crds *CRDWrapper) Resources() []*pipelineapi.PipelineResource {
	return crds.resources
}

// AddLabels merges the specified labels into the PipelineRun labels.
func (crds *CRDWrapper) AddLabels(labels map[string]string) {
	// only include labels on PipelineRuns because they're unique, Task and pipeline are static resources so we'd overwrite existing labels if applied to them too
	util.MergeMaps(crds.pipelineRun.Labels, labels)
}

// ObjectReferences creates the generic Kube resource metadata.
func (crds *CRDWrapper) ObjectReferences() []kube.ObjectReference {
	var resources []kube.ObjectReference
	for _, task := range crds.tasks {
		if task.ObjectMeta.Name == "" {
			log.Warnf("created Task has no name: %#v\n", task)

		} else {
			resources = append(resources, kube.CreateObjectReference(task.TypeMeta, task.ObjectMeta))
		}
	}
	if crds.pipeline != nil {
		if crds.pipeline.ObjectMeta.Name == "" {
			log.Warnf("created pipeline has no name: %#v\n", crds.pipeline)

		} else {
			resources = append(resources, kube.CreateObjectReference(crds.pipeline.TypeMeta, crds.pipeline.ObjectMeta))
		}
	}
	if crds.pipelineRun != nil {
		if crds.pipelineRun.ObjectMeta.Name == "" {
			log.Warnf("created pipelineRun has no name: %#v\n", crds.pipelineRun)
		} else {
			resources = append(resources, kube.CreateObjectReference(crds.pipelineRun.TypeMeta, crds.pipelineRun.ObjectMeta))
		}
	}
	if len(resources) == 0 {
		log.Warnf("no tasks, pipeline or PipelineRuns created\n")
	}
	return resources
}

// TODO: Use the same YAML lib here as in buildpipeline/pipeline.go
// TODO: Use interface{} with a helper function to reduce code repetition?
// TODO: Take no arguments and use o.Results internally?

// WriteToDisk writes the Tekton CRDs to disk. All CRDs are created in the specified directory. One YAML file per CRD.
func (crds *CRDWrapper) WriteToDisk(dir string, pipelineActivity *kube.PromoteStepActivityKey) error {
	if err := os.Mkdir(dir, os.ModePerm); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	data, err := yaml.Marshal(crds.pipeline)
	if err != nil {
		return errors.Wrap(err, "failed to marshal pipeline YAML")
	}
	fileName := filepath.Join(dir, "pipeline.yml")
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save pipeline file %s", fileName)
	}
	log.Infof("generated pipeline at %s\n", util.ColorInfo(fileName))

	data, err = yaml.Marshal(crds.pipelineRun)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal pipelineRun YAML")
	}
	fileName = filepath.Join(dir, "pipeline-run.yml")
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save pipelineRun file %s", fileName)
	}
	log.Infof("generated pipelineRun at %s\n", util.ColorInfo(fileName))

	if crds.structure != nil {
		data, err = yaml.Marshal(crds.structure)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal PipelineStructure YAML")
		}
		fileName = filepath.Join(dir, "structure.yml")
		err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save PipelineStructure file %s", fileName)
		}
		log.Infof("generated PipelineStructure at %s\n", util.ColorInfo(fileName))
	}

	taskList := &pipelineapi.TaskList{}
	for _, task := range crds.tasks {
		taskList.Items = append(taskList.Items, *task)
	}

	resourceList := &pipelineapi.PipelineResourceList{}
	for _, resource := range crds.resources {
		resourceList.Items = append(resourceList.Items, *resource)
	}

	data, err = yaml.Marshal(taskList)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal Task YAML")
	}
	fileName = filepath.Join(dir, "tasks.yml")
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save Task file %s", fileName)
	}
	log.Infof("generated Tasks at %s\n", util.ColorInfo(fileName))

	data, err = yaml.Marshal(resourceList)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal PipelineResource YAML")
	}
	fileName = filepath.Join(dir, "resources.yml")
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save PipelineResource file %s", fileName)
	}
	log.Infof("generated PipelineResources at %s\n", util.ColorInfo(fileName))

	data, err = yaml.Marshal(pipelineActivity)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal PipelineActivity YAML")
	}
	fileName = filepath.Join(dir, "pipelineActivity.yml")
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save PipelineActivity file %s", fileName)
	}
	log.Infof("generated PipelineActivity at %s\n", util.ColorInfo(fileName))

	return nil
}

// validates the resources of this wrapper
func (crds *CRDWrapper) validate() error {
	ctx := context.Background()
	if validateErr := crds.pipeline.Spec.Validate(ctx); validateErr != nil {
		return errors.Wrapf(validateErr, "validation failed for generated pipeline")
	}
	for _, task := range crds.tasks {
		if validateErr := task.Spec.Validate(ctx); validateErr != nil {
			data, _ := yaml.Marshal(task)
			return errors.Wrapf(validateErr, "validation failed for generated Task: %s %s", task.Name, string(data))
		}
	}

	if validateErr := crds.pipelineRun.Spec.Validate(ctx); validateErr != nil {
		return errors.Wrapf(validateErr, "validation for generated pipelineRun failed")
	}
	return nil
}
