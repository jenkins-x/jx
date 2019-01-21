package commoncmd

import (
	"strconv"
	"strings"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getAllPipelineJobNames returns all the pipeline job names
func (o *CommonOptions) GetAllPipelineJobNames(jenkinsClient gojenkins.JenkinsClient, jobNames *[]string, jobName string) error {
	job, err := jenkinsClient.GetJob(jobName)
	if err != nil {
		return err
	}
	if len(job.Jobs) == 0 {
		*jobNames = append(*jobNames, job.FullName)
	}
	for _, j := range job.Jobs {
		err = o.GetAllPipelineJobNames(jenkinsClient, jobNames, job.FullName+"/"+j.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *CommonOptions) GetPipelineName(gitInfo *gits.GitRepository, pipeline string, build string, appName string) (string, string) {
	if pipeline == "" {
		pipeline = o.GetJobName()
	}
	if build == "" {
		build = o.GetBuildNumber()
	}
	if gitInfo != nil && pipeline == "" {
		// lets default the pipeline name from the Git repo
		branch, err := o.Git().Branch(".")
		if err != nil {
			log.Warnf("Could not find the branch name: %s\n", err)
		}
		if branch == "" {
			branch = "master"
		}
		pipeline = util.UrlJoin(gitInfo.Organisation, gitInfo.Name, branch)
	}
	if pipeline == "" && appName != "" {
		suffix := appName + "/master"

		// lets try deduce the pipeline name via the app name
		jxClient, ns, err := o.JXClientAndDevNamespace()
		if err == nil {
			pipelineList, err := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
			if err == nil {
				for _, pipelineResource := range pipelineList.Items {
					pipelineName := pipelineResource.Spec.Pipeline
					if strings.HasSuffix(pipelineName, suffix) {
						pipeline = pipelineName
						break
					}
				}
			}
		}
	}
	if pipeline == "" {
		// lets try find
		log.Warnf("No $JOB_NAME environment variable found so cannot record promotion activities into the PipelineActivity resources in kubernetes\n")
	} else if build == "" {
		// lets validate and determine the current active pipeline branch
		p, b, err := o.getLatestPipelineBuild(pipeline)
		if err != nil {
			log.Warnf("Failed to try detect the current Jenkins pipeline for %s due to %s\n", pipeline, err)
			build = "1"
		} else {
			pipeline = p
			build = b
		}
	}
	return pipeline, build
}

// GetLatestPipelineBuild for the given pipeline name lets try find the Jenkins Pipeline and the latest build
func (o *CommonOptions) getLatestPipelineBuild(pipeline string) (string, string, error) {
	log.Infof("pipeline %s\n", pipeline)
	build := ""
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return pipeline, build, err
	}
	kubeClient, err := o.KubeClient()
	if err != nil {
		return pipeline, build, err
	}
	devEnv, err := kube.GetEnrichedDevEnvironment(kubeClient, jxClient, ns)
	webhookEngine := devEnv.Spec.WebHookEngine
	if webhookEngine == v1.WebHookEngineProw {
		return pipeline, build, nil
	}

	jenkins, err := o.JenkinsClient()
	if err != nil {
		return pipeline, build, err
	}
	paths := strings.Split(pipeline, "/")
	job, err := jenkins.GetJobByPath(paths...)
	if err != nil {
		return pipeline, build, err
	}
	build = strconv.Itoa(job.LastBuild.Number)
	return pipeline, build, nil
}
