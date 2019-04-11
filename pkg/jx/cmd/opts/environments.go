package opts

import (
	"fmt"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RegisterEnvironmentCRD registers the CRD for environmnt
func (o *CommonOptions) RegisterEnvironmentCRD() error {
	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterEnvironmentCRD(apisClient)
	return err
}

// ModifyDevEnvironment performs some mutation on the Development environment to modify team settings
func (o *CommonOptions) ModifyDevEnvironmentWithNs(jxClient versioned.Interface, ns string,
	fn func(env *jenkinsv1.Environment) error) error {
	env, err := kube.EnsureDevEnvironmentSetup(jxClient, ns)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure that dev environment is setup for namespace '%s'", ns)
	}
	if env == nil {
		return fmt.Errorf("No Development environment found in namespace %s", ns)
	}
	err = fn(env)
	if err != nil {
		return errors.Wrap(err, "failed to call the callback function for dev environment")
	}
	_, err = jxClient.JenkinsV1().Environments(ns).PatchUpdate(env)
	if err != nil {
		return fmt.Errorf("Failed to update Development environment in namespace %s: %s", ns, err)
	}
	return nil
}

// GetDevEnv gets the Development Environment CRD as devEnv,
// and also tells the user whether the development environment is using gitOps
func (o *CommonOptions) GetDevEnv() (gitOps bool, devEnv *jenkinsv1.Environment) {
	// We're going to need to know whether the team is using GitOps for the dev env or not,
	// and also access the team settings, so load those
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		if o.Verbose {
			logrus.Errorf("Error loading team settings. %v\n", err)
		}
		return false, &jenkinsv1.Environment{}
	} else {
		devEnv, err := kube.GetDevEnvironment(jxClient, ns)
		if err != nil {
			logrus.Errorf("Error loading team settings. %v\n", err)
			return false, &jenkinsv1.Environment{}
		}
		gitOps := false
		if devEnv == nil {
			devEnv = &jenkinsv1.Environment{}
			devEnv.Spec.Namespace = ns
		}
		if devEnv.Spec.Source.URL != "" {
			gitOps = true
		}
		return gitOps, devEnv
	}
}
