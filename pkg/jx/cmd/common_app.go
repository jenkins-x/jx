package cmd

import (
	"fmt"
	"github.com/pkg/errors"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/extensions"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetDevEnv gets the Development Enviornment CRD as devEnv,
// and also tells the user whether the development environment is using gitOps
func (o *CommonOptions) GetDevEnv() (gitOps bool, devEnv *jenkinsv1.Environment) {

	// We're going to need to know whether the team is using GitOps for the dev env or not,
	// and also access the team settings, so load those
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		if o.Verbose {
			log.Errorf("Error loading team settings. %v\n", err)
		}
		return false, &jenkinsv1.Environment{}
	} else {
		devEnv, err := kube.GetDevEnvironment(jxClient, ns)
		if err != nil {
			log.Errorf("Error loading team settings. %v\n", err)
			return false, &jenkinsv1.Environment{}
		}
		gitOps := false
		if devEnv.Spec.Source.URL != "" {
			gitOps = true
		}
		return gitOps, devEnv
	}
}

// OnAppInstall calls extensions.OnAppInstall for the current cmd, passing app and version
func (o *CommonOptions) OnAppInstall(app string, version string) error {
	// Find the app metadata, if any
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	certClient, err := o.CreateCertManagerClient()
	if err != nil {
		return err
	}
	selector := fmt.Sprintf("chart=%s-%s", app, version)
	appList, err := jxClient.JenkinsV1().Apps(ns).List(metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return err
	}
	if len(appList.Items) > 1 {
		return fmt.Errorf("more than one app (%v) was found for %s", appList.Items, selector)
	} else if len(appList.Items) == 1 {
		versionsDir, err := o.cloneJXVersionsRepo("")
		if err != nil {
			return errors.Wrapf(err, "failed to clone the Jenkins X versions repository")
		}
		return extensions.OnInstallFromName(app, jxClient, kubeClient, certClient, ns, o.Helm(), defaultInstallTimeout, versionsDir)
	}
	return nil
}
