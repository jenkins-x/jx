package github

import (
	"fmt"
	"os"
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/github"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/tenant"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StepGithubAppTokenOptions contains the command line flags
type StepGithubAppTokenOptions struct {
	step.StepOptions
	GithubOrg         string
	Dir               string
	TenantServiceUrl  string
	TenantServiceAuth string
	GithubAppUrl      string
}

var (
	stepGithubAppTokenLong = templates.LongDesc(`
		This step requests the installation token from the tenant service and then calls the github app to get an installation token.
	`)

	stepGitHubAppTokenExample = templates.Examples(`
		jx step github app token
	`)
)

// NewCmdStepGithubAppToken calls the Jenkins-X github app to get an access token
func NewCmdStepGithubAppToken(commonOpts *opts.CommonOptions) *cobra.Command {

	options := StepGithubAppTokenOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "token",
		Short:   "requests an installation token from the github app",
		Long:    stepGithubAppTokenLong,
		Example: stepGitHubAppTokenExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.GithubOrg, "org", "o", "", "the github organisation the app has been installed into")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the values.yaml file")
	cmd.Flags().StringVarP(&options.TenantServiceUrl, "tenantServiceUrl", "t", "", "the tenant service url")
	cmd.Flags().StringVarP(&options.TenantServiceAuth, "tenantServiceAuth", "a", "", "the tenant service auth")
	cmd.Flags().StringVarP(&options.GithubAppUrl, "githubAppUrl", "g", "", "the github app url")

	return cmd
}

// Run implements this command
func (options *StepGithubAppTokenOptions) Run() error {

	requirements, requirementsFileName, err := config.LoadRequirementsConfig(options.Dir)
	if err != nil {
		log.Logger().Error("Unable to find requirements file %q", requirementsFileName)
		return errors.Wrapf(err, "failed to load Jenkins X requirements")
	}

	if options.GithubOrg == "" && requirements.Cluster.EnvironmentGitOwner != "" {
		options.GithubOrg = requirements.Cluster.EnvironmentGitOwner
		log.Logger().Debugf("github organisation is %s", options.GithubOrg)
	}

	if options.GithubOrg == "" {
		return errors.New("unable to find github organisation")
	}

	if options.TenantServiceUrl == "" && requirements.Ingress.DomainIssuerURL != "" {
		options.TenantServiceUrl = requirements.Ingress.DomainIssuerURL
		log.Logger().Debugf("tenant service url %q", options.TenantServiceUrl)

	}
	if options.TenantServiceUrl == "" {
		return errors.New("unable to find tenant service url")
	}

	if options.TenantServiceAuth == "" {
		options.TenantServiceAuth, err = options.getTenantServiceBasicAuth()
		if err != nil {
			log.Logger().Error("unable to get tenant service auth")
			return errors.Wrapf(err, "unable to get tenant service auth")
		}
	}

	if options.TenantServiceAuth == "" {
		return errors.New("Unable to get tenant service auth")
	}

	installationId, err := options.getInstallationId()
	if err != nil {
		log.Logger().Error("Unable to get installation id")
		return errors.Wrapf(err, "Unable to get installation id")
	}
	log.Logger().Debugf("installationId is %s\n", installationId)

	installationToken, err := options.getInstallationToken(installationId)
	if err != nil {
		log.Logger().Error("Unable to get installation token")
		return errors.Wrapf(err, "Unable to get installation token")
	}

	err = options.createSecret(installationToken, requirements)
	if err != nil {
		log.Logger().Error("Error writing secret")
		return err
	}

	log.Logger().Debugf("installationToken is %s\n", installationToken)
	return err
}

func (options *StepGithubAppTokenOptions) createSecret(token string, requirements *config.RequirementsConfig) error {

	var gitKind, gitName, gitServer, namespace string
	if requirements.Cluster.GitKind != "" {
		gitKind = requirements.Cluster.GitKind
	} else {
		gitKind = "github"
	}

	if requirements.Cluster.GitName != "" {
		gitName = requirements.Cluster.GitName
	} else {
		gitName = "github"
	}

	if requirements.Cluster.GitServer != "" {
		gitServer = requirements.Cluster.GitServer
	} else {
		gitServer = "https://github.com"
	}
	name := "jx-pipeline-git"
	name = strings.Join([]string{name, gitKind, gitName}, "-")

	log.Logger().Debugf("Creating or updating secret %s", name)

	labels := map[string]string{
		"jenkins.io/service-kind":     gitKind,
		"jenkins.io/created-by":       "jx",
		"jenkins.io/credentials-type": "usernamePassword",
		"jenkins.io/kind":             "git",
	}

	annotations := map[string]string{
		"jenkins.io/url":                     gitServer,
		"jenkins.io/name":                    gitName,
		"jenkins.io/credentials-description": "API Token for acccessing " + gitServer + " Git service inside pipelines",
		"build.knative.dev/git-0":            gitServer,
	}

	data := map[string][]byte{
		"password": []byte(token),
		"username": []byte("jenkins-x[bot]"),
	}

	k8Secret := &corev1.Secret{ //pragma: allowlist secret
		ObjectMeta: metav1.ObjectMeta{
			Name:                       name,
			DeletionGracePeriodSeconds: nil,
			Labels:                     labels,
			Annotations:                annotations,
		},
		Data: data,
	}

	kubeClient, err := options.KubeClient()
	if err != nil {
		log.Logger().Errorf("error getting kube client %v", err)
		return err
	}

	if requirements.Cluster.Namespace != "" {
		namespace = requirements.Cluster.Namespace
	} else {
		namespace = "jx"
	}

	coreV1 := kubeClient.CoreV1()
	secretInterface := coreV1.Secrets(namespace)
	currentSecret, err := secretInterface.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Logger().Errorf("error getting secret %v", err)
	} else if currentSecret != nil {
		_, err = secretInterface.Update(k8Secret)
		if err != nil {
			log.Logger().Errorf("error updating secret %v", err)
		}
		log.Logger().Debugf("Secret %s updated", name)
	} else {
		_, err = secretInterface.Create(k8Secret)
		if err != nil {
			log.Logger().Errorf("error creating secret %v", err)
		}
		log.Logger().Debugf("Secret %s created", name)
	}

	return err
}

func (options *StepGithubAppTokenOptions) getTenantServiceBasicAuth() (string, error) {
	username := os.Getenv(config.RequirementDomainIssuerUsername)
	password := os.Getenv(config.RequirementDomainIssuerPassword)

	if username == "" {
		return "", errors.Errorf("no %s environment variable found", config.RequirementDomainIssuerUsername)
	}
	if password == "" {
		return "", errors.Errorf("no %s environment variable found", config.RequirementDomainIssuerPassword)
	}

	tenantServiceAuth := fmt.Sprintf("%s:%s", username, password)
	return tenantServiceAuth, nil
}

func (options *StepGithubAppTokenOptions) getInstallationId() (string, error) {
	org := options.GithubOrg
	log.Logger().Debugf("github org %s", org)
	tenantServiceClient := tenant.NewTenantClient()

	installationID, err := tenantServiceClient.GetInstallationID(options.TenantServiceUrl, options.TenantServiceAuth, org)
	if err != nil {
		log.Logger().Errorf("error calling tenant service %v", err)
		return "", err
	}

	log.Logger().Debugf("installation id %q\n", installationID)
	return installationID, nil
}

func (options *StepGithubAppTokenOptions) getInstallationToken(installationID string) (string, error) {
	org := options.GithubOrg
	log.Logger().Debugf("github org %s\n", org)

	installationToken, err := github.GetInstallationToken(options.GithubAppUrl, installationID)
	if err != nil {
		log.Logger().Errorf("error calling github app %v", err)
		return "", err
	}

	log.Logger().Debugf("installation token %q", installationToken)
	return installationToken.InstallationToken, nil
}
