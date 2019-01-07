package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
)

func (o *CommonOptions) CreateGitAuthConfigServiceDryRun(dryRun bool) (auth.ConfigService, error) {
	if dryRun {
		fileName := GitAuthConfigFile
		return o.CreateGitAuthConfigServiceFromSecrets(fileName, nil, false)
	}
	return o.CreateGitAuthConfigService()
}

func (o *CommonOptions) CreateGitAuthConfigService() (auth.ConfigService, error) {
	var secrets *corev1.SecretList
	var err error
	if !o.SkipAuthSecretsMerge {
		secrets, err = o.LoadPipelineSecrets(kube.ValueKindGit, "")
		if err != nil {

			kubeConfig, _, configLoadErr := o.Kube().LoadConfig()
			if configLoadErr != nil {
				log.Warnf("WARNING: Could not load config: %s", configLoadErr)
			}

			ns := kube.CurrentNamespace(kubeConfig)
			if ns == "" {
				log.Warnf("WARNING: Could not get the current namespace")
			}

			log.Warnf("WARNING: The current user cannot query secrets in the namespace %s: %s\n", ns, err)
		}
	}

	fileName := GitAuthConfigFile
	return o.CreateGitAuthConfigServiceFromSecrets(fileName, secrets, o.IsInCDPipeline())
}

// CreateGitAuthConfigServiceFromSecrets Creates a git auth config service from secrets
func (o *CommonOptions) CreateGitAuthConfigServiceFromSecrets(fileName string, secrets *corev1.SecretList, isCDPipeline bool) (auth.ConfigService, error) {
	authConfigSvc, err := o.CreateAuthConfigService(fileName)
	if err != nil {
		return authConfigSvc, err
	}

	config, err := authConfigSvc.LoadConfig()
	if err != nil {
		return authConfigSvc, err
	}

	if secrets != nil {
		err = o.AuthMergePipelineSecrets(config, secrets, kube.ValueKindGit, isCDPipeline || o.IsInCluster())
		if err != nil {
			return authConfigSvc, err
		}
	}

	// lets add a default if there's none defined yet
	if len(config.Servers) == 0 {
		// if in cluster then there's no user configfile, so check for env vars first
		userAuth := auth.CreateAuthUserFromEnvironment("GIT")

		if !userAuth.IsInvalid() {
			// if no config file is being used lets grab the git server from the current directory
			server, err := o.Git().Server("")
			if err != nil {
				log.Warnf("WARNING: unable to get remote Git repo server, %v\n", err)
				server = "https://github.com"
			}
			config.Servers = []*auth.AuthServer{
				{
					Name:  "Git",
					URL:   server,
					Users: []*auth.UserAuth{&userAuth},
				},
			}
		}
	}

	if len(config.Servers) == 0 {
		config.Servers = []*auth.AuthServer{
			{
				Name:  "GitHub",
				URL:   "https://github.com",
				Kind:  gits.KindGitHub,
				Users: []*auth.UserAuth{},
			},
		}
	}

	return authConfigSvc, nil
}

func (o *CommonOptions) LoadPipelineSecrets(kind, serviceKind string) (*corev1.SecretList, error) {
	// TODO return empty list if not inside a pipeline?
	kubeClient, curNs, err := o.KubeClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to create a Kubernetes client %s", err)
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, curNs)
	if err != nil {
		return nil, fmt.Errorf("Failed to get the development environment %s", err)
	}

	var selector string
	if kind != "" {
		selector = kube.LabelKind + "=" + kind
	}
	if serviceKind != "" {
		selector = kube.LabelServiceKind + "=" + serviceKind
	}

	opts := metav1.ListOptions{
		LabelSelector: selector,
	}
	return kubeClient.CoreV1().Secrets(ns).List(opts)
}
