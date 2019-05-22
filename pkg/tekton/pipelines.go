package tekton

import (
	"fmt"
	jenkinsio "github.com/jenkins-x/jx/pkg/apis/jenkins.io"
	"reflect"
	"strconv"
	"time"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	jxClient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GeneratePipelineActivity generates a initial PipelineActivity CRD so UI/get act can get an earlier notification that the jobs have been scheduled
func GeneratePipelineActivity(buildNumber string, branch string, gitInfo *gits.GitRepository) *kube.PromoteStepActivityKey {
	name := gitInfo.Organisation + "-" + gitInfo.Name + "-" + branch + "-" + buildNumber
	pipeline := gitInfo.Organisation + "/" + gitInfo.Name + "/" + branch
	log.Infof("PipelineActivity for %s", name)
	return &kube.PromoteStepActivityKey{
		PipelineActivityKey: kube.PipelineActivityKey{
			Name:     name,
			Pipeline: pipeline,
			Build:    buildNumber,
			GitInfo:  gitInfo,
		},
	}
}

// CreateOrUpdateSourceResource lazily creates a Tekton Pipeline PipelineResource for the given git repository
func CreateOrUpdateSourceResource(tektonClient tektonclient.Interface, ns string, created *v1alpha1.PipelineResource) (*v1alpha1.PipelineResource, error) {
	resourceName := created.Name
	resourceInterface := tektonClient.TektonV1alpha1().PipelineResources(ns)

	_, err := resourceInterface.Create(created)
	if err == nil {
		return created, nil
	}

	answer, err2 := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err2 != nil {
		return answer, errors.Wrapf(err, "failed to get PipelineResource %s with %v after failing to create a new one", resourceName, err2)
	}
	if !reflect.DeepEqual(&created.Spec, &answer.Spec) {
		answer.Spec = created.Spec
		answer, err = resourceInterface.Update(answer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to update PipelineResource %s", resourceName)
		}
	}
	return answer, nil
}

// CreateOrUpdateTask lazily creates a Tekton Pipeline Task
func CreateOrUpdateTask(tektonClient tektonclient.Interface, ns string, created *v1alpha1.Task) (*v1alpha1.Task, error) {
	resourceName := created.Name
	if resourceName == "" {
		return nil, fmt.Errorf("the Task must have a name")
	}
	resourceInterface := tektonClient.TektonV1alpha1().Tasks(ns)

	_, err := resourceInterface.Create(created)
	if err == nil {
		return created, nil
	}

	answer, err2 := resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err2 != nil {
		return answer, errors.Wrapf(err, "failed to get PipelineResource %s with %v after failing to create a new one", resourceName, err2.Error())
	}
	if !reflect.DeepEqual(&created.Spec, &answer.Spec) || !reflect.DeepEqual(created.Annotations, answer.Annotations) || !reflect.DeepEqual(created.Labels, answer.Labels) {
		answer.Spec = created.Spec
		answer.Labels = util.MergeMaps(answer.Labels, created.Labels)
		answer.Annotations = util.MergeMaps(answer.Annotations, created.Annotations)
		answer, err = resourceInterface.Update(answer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to update PipelineResource %s", resourceName)
		}
	}
	return answer, nil
}

// GenerateNextBuildNumber generates a new build number for the given project.
func GenerateNextBuildNumber(tektonClient tektonclient.Interface, jxClient jxClient.Interface, ns string, gitInfo *gits.GitRepository, branch string, duration time.Duration, pipelineIdentifier string) (string, error) {
	nextBuildNumber := ""
	resourceInterface := jxClient.JenkinsV1().SourceRepositories(ns)
	// TODO: How does SourceRepository handle name overlap?
	sourceRepoName := kube.ToValidName(gitInfo.Organisation + "-" + gitInfo.Name)

	f := func() error {
		sourceRepo, err := kube.GetOrCreateSourceRepository(jxClient, ns, gitInfo.Name, gitInfo.Organisation, gitInfo.ProviderURL())
		if err != nil {
			return errors.Wrapf(err, "Unable to generate next build number for %s/%s", sourceRepoName, branch)
		}
		sourceRepoName = sourceRepo.Name
		if sourceRepo.Annotations == nil {
			sourceRepo.Annotations = make(map[string]string, 1)
		}
		annKey := LastBuildNumberAnnotationPrefix + kube.ToValidName(branch)
		annVal := sourceRepo.Annotations[annKey]
		lastBuildNumber := 0
		if annVal != "" {
			lastBuildNumber, err = strconv.Atoi(annVal)
			if err != nil {
				return errors.Wrapf(err, "Expected number but SourceRepository %s has annotation %s with value %s\n", sourceRepoName, annKey, annVal)
			}
		}
		for nextNumber := lastBuildNumber + 1; true; nextNumber++ {
			// lets check there is not already a PipelineRun for this number
			buildIdentifier := strconv.Itoa(nextNumber)
			pipelineResourceName := syntax.PipelineRunName(pipelineIdentifier, buildIdentifier)
			_, err := tektonClient.TektonV1alpha1().PipelineRuns(ns).Get(pipelineResourceName, metav1.GetOptions{})
			if err == nil {
				// lets try make another build number as there's already a PipelineRun
				// which could be due to name clashes
				continue
			}
			sourceRepo.Annotations[annKey] = buildIdentifier
			if _, err := resourceInterface.Update(sourceRepo); err != nil {
				return err
			}
			nextBuildNumber = sourceRepo.Annotations[annKey]
			return nil
		}
		return nil
	}

	err := util.Retry(duration, f)
	if err != nil {
		return "", err
	}
	return nextBuildNumber, nil
}

// GenerateSourceRepoResource generates the PipelineResource for the git repository we are operating on.
func GenerateSourceRepoResource(name string, gitInfo *gits.GitRepository, revision string) *pipelineapi.PipelineResource {
	if gitInfo == nil || gitInfo.HttpsURL() == "" {
		return nil

	}

	resource := &pipelineapi.PipelineResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: syntax.TektonAPIVersion,
			Kind:       "PipelineResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: pipelineapi.PipelineResourceSpec{
			Type: pipelineapi.PipelineResourceTypeGit,
			Params: []pipelineapi.Param{
				{
					Name:  "revision",
					Value: revision,
				},
				{
					Name:  "url",
					Value: gitInfo.HttpsURL(),
				},
			},
		},
	}

	return resource
}

// CreatePipelineRun creates the PipelineRun struct.
func CreatePipelineRun(resources []*pipelineapi.PipelineResource,
	name string,
	apiVersion string,
	labels map[string]string,
	trigger string,
	serviceAccount string,
	pipelineParams []pipelineapi.Param) *pipelineapi.PipelineRun {
	var resourceBindings []pipelineapi.PipelineResourceBinding
	for _, resource := range resources {
		resourceBindings = append(resourceBindings, pipelineapi.PipelineResourceBinding{
			Name: resource.Name,
			ResourceRef: pipelineapi.PipelineResourceRef{
				Name:       resource.Name,
				APIVersion: resource.APIVersion,
			},
		})
	}

	pipelineRun := &pipelineapi.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: syntax.TektonAPIVersion,
			Kind:       "PipelineRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: util.MergeMaps(labels),
		},
		Spec: pipelineapi.PipelineRunSpec{
			ServiceAccount: serviceAccount,
			Trigger: pipelineapi.PipelineTrigger{
				Type: pipelineapi.PipelineTriggerType(trigger),
			},
			PipelineRef: pipelineapi.PipelineRef{
				Name:       name,
				APIVersion: apiVersion,
			},
			Resources: resourceBindings,
			Params:    pipelineParams,
		},
	}

	return pipelineRun
}

// ApplyPipelineRun lazily creates a Tekton PipelineRun.
func ApplyPipelineRun(tektonClient tektonclient.Interface, ns string, run *v1alpha1.PipelineRun) (*v1alpha1.PipelineRun, error) {
	resourceName := run.Name
	resourceInterface := tektonClient.TektonV1alpha1().PipelineRuns(ns)

	answer, err := resourceInterface.Create(run)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create PipelineRun %s", resourceName)
	}
	return answer, nil
}

// CreateOrUpdatePipeline lazily creates a Tekton Pipeline for the given git repository, branch and context
func CreateOrUpdatePipeline(tektonClient tektonclient.Interface, ns string, created *v1alpha1.Pipeline) (*v1alpha1.Pipeline, error) {
	resourceName := created.Name
	resourceInterface := tektonClient.TektonV1alpha1().Pipelines(ns)

	answer, err := resourceInterface.Create(created)
	if err == nil {
		return answer, nil
	}

	answer, err = resourceInterface.Get(resourceName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to get Pipeline %s after failing to create a new one", resourceName)
	}

	if !reflect.DeepEqual(&created.Spec, &answer.Spec) || !reflect.DeepEqual(created.Labels, answer.Labels) {
		answer.Annotations = util.MergeMaps(answer.Annotations, created.Annotations)
		answer.Spec = created.Spec
		answer, err = resourceInterface.Update(answer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to update Pipeline %s", resourceName)
		}
	}
	return answer, nil
}

// PipelineResourceName returns the pipeline resource name for the given git repository, branch and context
func PipelineResourceName(gitInfo *gits.GitRepository, branch string, context string) string {
	organisation := gitInfo.Organisation
	name := gitInfo.Name
	dirtyName := organisation + "-" + name + "-" + branch
	if context != "" {
		dirtyName += "-" + context
	}
	// TODO: https://github.com/tektoncd/pipeline/issues/481 causes
	// problems since autogenerated container names can end up surpassing 63
	// characters, which is not allowed. Longest known prefix for now is 28
	// chars (build-step-artifact-copy-to-), so we truncate to 35 so the
	// generated container names are no more than 63 chars.
	resourceName := kube.ToValidNameTruncated(dirtyName, 31)
	return resourceName
}

// ApplyPipeline applies the tasks and pipeline to the cluster
// and creates and applies a PipelineResource for their source repo and a pipelineRun
// to execute them.
func ApplyPipeline(jxClient versioned.Interface, tektonClient tektonclient.Interface, crds *CRDWrapper, ns string, gitInfo *gits.GitRepository, branch string, activityKey *kube.PromoteStepActivityKey) error {
	info := util.ColorInfo

	var activityOwnerReference *metav1.OwnerReference

	if activityKey != nil {
		activity, _, err := activityKey.GetOrCreate(jxClient, crds.Pipeline().Namespace)
		if err != nil {
			return err
		}

		activityOwnerReference = &metav1.OwnerReference{
			APIVersion: jenkinsio.GroupAndVersion,
			Kind:       "PipelineActivity",
			Name:       activity.Name,
			UID:        activity.UID,
		}
	}

	for _, resource := range crds.Resources() {
		_, err := CreateOrUpdateSourceResource(tektonClient, ns, resource)
		if err != nil {
			return errors.Wrapf(err, "failed to create/update PipelineResource %s in namespace %s", resource.Name, ns)
		}
		if resource.Spec.Type == pipelineapi.PipelineResourceTypeGit {
			gitURL := gitInfo.HttpCloneURL()
			log.Infof("upserted PipelineResource %s for the git repository %s and branch %s\n", info(resource.Name), info(gitURL), info(branch))
		} else {
			log.Infof("upserted PipelineResource %s\n", info(resource.Name))
		}
	}

	for _, task := range crds.Tasks() {
		if activityOwnerReference != nil {
			task.OwnerReferences = []metav1.OwnerReference{*activityOwnerReference}
		}
		_, err := CreateOrUpdateTask(tektonClient, ns, task)
		if err != nil {
			return errors.Wrapf(err, "failed to create/update the task %s in namespace %s", task.Name, ns)
		}
		log.Infof("upserted Task %s\n", info(task.Name))
	}

	if activityOwnerReference != nil {
		crds.Pipeline().OwnerReferences = []metav1.OwnerReference{*activityOwnerReference}
	}

	pipeline, err := CreateOrUpdatePipeline(tektonClient, ns, crds.Pipeline())
	if err != nil {
		return errors.Wrapf(err, "failed to create/update the pipeline in namespace %s", ns)
	}
	log.Infof("upserted Pipeline %s\n", info(pipeline.Name))

	pipelineOwnerReference := metav1.OwnerReference{
		APIVersion: syntax.TektonAPIVersion,
		Kind:       "pipeline",
		Name:       pipeline.Name,
		UID:        pipeline.UID,
	}

	crds.structure.OwnerReferences = []metav1.OwnerReference{pipelineOwnerReference}
	crds.pipelineRun.OwnerReferences = []metav1.OwnerReference{pipelineOwnerReference}

	_, err = ApplyPipelineRun(tektonClient, ns, crds.PipelineRun())
	if err != nil {
		return errors.Wrapf(err, "failed to create the pipelineRun in namespace %s", ns)
	}
	log.Infof("created PipelineRun %s\n", info(crds.PipelineRun().Name))

	if crds.Structure() != nil {
		crds.Structure().PipelineRunRef = &crds.PipelineRun().Name

		structuresClient := jxClient.JenkinsV1().PipelineStructures(ns)

		// Reset the structure name to be the run's name and set the PipelineRef and PipelineRunRef
		if crds.Structure().PipelineRef == nil {
			crds.Structure().PipelineRef = &pipeline.Name
		}
		crds.Structure().Name = crds.PipelineRun().Name
		crds.Structure().PipelineRunRef = &crds.PipelineRun().Name

		if _, structErr := structuresClient.Create(crds.Structure()); structErr != nil {
			return errors.Wrapf(structErr, "failed to create the PipelineStructure in namespace %s", ns)
		}
		log.Infof("created PipelineStructure %s\n", info(crds.Structure().Name))
	}

	return nil
}
