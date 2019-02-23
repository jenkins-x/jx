package cmd

import (
	"fmt"
	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sort"
)

// JenkinsSelectorOptions used to represent the options used to refer to a Jenkins.
// if nothing is specified it assumes the current team is using a static Jenkins server as its execution engine.
// otherwise we can refer to other additional Jenkins Apps to implement custom Jenkins servers
type JenkinsSelectorOptions struct {
	UseCustomJenkins  bool
	CustomJenkinsName string

	// cached client
	cachedCustomJenkinsClient gojenkins.JenkinsClient
}

// AddFlags add the command flags for picking a custom Jenkins App to work with
func (o *JenkinsSelectorOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&o.UseCustomJenkins, "custom", "m", false, "Use a custom Jenkins App instead of the default execution engine in Jenkins X")
	cmd.Flags().StringVarP(&o.CustomJenkinsName, "jenkins-name", "j", "", "The name of the custom Jenkins App if you don't wish to use the default execution engine in Jenkins X")
}

// IsCustom returns true if a custom Jenkins App is specified
func (o *JenkinsSelectorOptions) IsCustom() bool {
	return o.UseCustomJenkins || o.CustomJenkinsName != ""
}

// getAllPipelineJobNames returns all the pipeline job names
func (o *CommonOptions) getAllPipelineJobNames(jenkinsClient gojenkins.JenkinsClient, jobNames *[]string, jobName string) error {
	job, err := jenkinsClient.GetJob(jobName)
	if err != nil {
		return err
	}
	if len(job.Jobs) == 0 {
		*jobNames = append(*jobNames, job.FullName)
	}
	for _, j := range job.Jobs {
		err = o.getAllPipelineJobNames(jenkinsClient, jobNames, job.FullName+"/"+j.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetJenkinsClient sets the JenkinsClient - usually used in testing
func (o *CommonOptions) SetJenkinsClient(jenkinsClient gojenkins.JenkinsClient) {
	o.jenkinsClient = jenkinsClient
}

// JenkinsClient returns the Jenkins client
func (o *CommonOptions) JenkinsClient() (gojenkins.JenkinsClient, error) {
	if o.jenkinsClient == nil {
		kubeClient, ns, err := o.KubeClientAndDevNamespace()
		if err != nil {
			return nil, err
		}

		o.factory.SetBatch(o.BatchMode)
		jenkins, err := o.factory.CreateJenkinsClient(kubeClient, ns, o.In, o.Out, o.Err)

		if err != nil {
			return nil, err
		}
		o.jenkinsClient = jenkins
	}
	return o.jenkinsClient, nil
}

// CustomJenkinsClient returns the Jenkins client for the custom jenkins app
func (o *CommonOptions) CustomJenkinsClient(jenkinsServiceName string) (gojenkins.JenkinsClient, error) {
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, err
	}
	o.factory.SetBatch(o.BatchMode)
	return o.factory.CreateCustomJenkinsClient(kubeClient, ns, jenkinsServiceName, o.In, o.Out, o.Err)
}

// CustomJenkinsURL returns the default or the custom Jenkins URL
func (o *CommonOptions) CustomJenkinsURL(jenkinsSelector *JenkinsSelectorOptions, kubeClient kubernetes.Interface, ns string) (string, error) {
	if !jenkinsSelector.UseCustomJenkins {
		return o.factory.GetJenkinsURL(kubeClient, ns)
	}
	customJenkinsName, err := o.PickCustomJenkinsName(jenkinsSelector, kubeClient, ns)
	if err != nil {
		return "", err
	}
	return o.factory.GetCustomJenkinsURL(kubeClient, ns, customJenkinsName)
}

// PickCustomJenkinsName picks the name of a custom jenkins server App if available
func (o *CommonOptions) PickCustomJenkinsName(jenkinsSelector *JenkinsSelectorOptions, kubeClient kubernetes.Interface, ns string) (string, error) {
	if !jenkinsSelector.UseCustomJenkins {
		return "", nil
	}
	customJenkinsName := jenkinsSelector.CustomJenkinsName
	if customJenkinsName == "" {
		serviceInterface := kubeClient.CoreV1().Services(ns)
		selector := kube.LabelKind + "=" + kube.ValueKindJenkins
		serviceList, err := serviceInterface.List(metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return "", errors.Wrapf(err, "failed to list Jenkins services in namespace %s with selector %s", ns, selector)
		}
		switch len(serviceList.Items) {
		case 0:
			return "", fmt.Errorf("No Jenkins App services found in namespace %s with selector %s\nAre you sure you installed a Jenkins App in this namespace?\nTry jx add app jx-app-jenkins", ns, selector)

		case 1:
			customJenkinsName = serviceList.Items[0].Name

		default:
			names := []string{}
			for _, svc := range serviceList.Items {
				names = append(names, svc.Name)
			}
			sort.Strings(names)

			if o.BatchMode {
				return "", util.MissingOptionWithOptions("jenkins-name", names)
			}
			customJenkinsName, err = util.PickName(names, "Pick which custom Jenkins App you wish to use: ", "Jenkins Apps are a way to add custom Jenkins servers into Jenkins X", o.GetIn(), o.GetOut(), o.GetErr())
			if err != nil {
				return "", err
			}
		}
	}
	jenkinsSelector.CustomJenkinsName = customJenkinsName
	if customJenkinsName == "" {
		return "", fmt.Errorf("failed to find a csutom Jenkins App name in namespace %s", ns)
	}
	return customJenkinsName, nil
}

// CreateCustomJenkinsClient creates either a regular Jenkins client or if useCustom is true creates a JenkinsClient
// for a custom jenkins App. If no customJenkinsName is specified and there is only one available it is used. Otherwise
// the user is prompted to pick the Jenkins App to use if not in batch mode.
func (o *CommonOptions) CreateCustomJenkinsClient(jenkinsSelector *JenkinsSelectorOptions) (gojenkins.JenkinsClient, error) {
	if jenkinsSelector == nil || !jenkinsSelector.UseCustomJenkins {
		return o.JenkinsClient()
	}
	if jenkinsSelector.cachedCustomJenkinsClient != nil {
		return jenkinsSelector.cachedCustomJenkinsClient, nil
	}
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, err
	}
	customJenkinsName, err := o.PickCustomJenkinsName(jenkinsSelector, kubeClient, ns)
	if err != nil {
		return nil, err
	}
	jenkinsClient, err := o.CustomJenkinsClient(customJenkinsName)
	if err == nil {
		jenkinsSelector.cachedCustomJenkinsClient = jenkinsClient
	}
	return jenkinsClient, err
}

// getJenkinsURL return the Jenkins URL
func (o *CommonOptions) getJenkinsURL() (string, error) {
	kubeClient, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return "", err
	}

	return o.factory.GetJenkinsURL(kubeClient, ns)
}
