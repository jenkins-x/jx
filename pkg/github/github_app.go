package github

import (
	"os"
	"strings"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type GithubApp struct {
	KubeClient kubernetes.Interface
}

func (githubApp *GithubApp) IsGitHubAppEnabledForOrganisation(organisation string) bool {
	for _, owner := range githubApp.GetGitHubAppOwners() {
		if owner == organisation {
			return true
		}
	}
	return false
}

func (githubApp *GithubApp) GetGitHubAppOwners() []string {

	githubApps := make([]string, 0)
	kubeClient := githubApp.KubeClient
	namespace := os.Getenv(config.BootDeployNamespace)

	if namespace == "" {
		namespace = "jx"
	}

	secretsInterface := kubeClient.CoreV1().Secrets(namespace)

	selector := kube.LabelKind + "=git"

	options := metav1.ListOptions{
		LabelSelector: selector,
	}

	secretsList, err := secretsInterface.List(options)
	if err != nil {
		log.Logger().Errorf("error listing secrets")
		return githubApps
	}

	for _, s := range secretsList.Items {
		url := s.Annotations["jenkins.io/url"]
		if isGithubAppUrl(url) {
			githubApps = append(githubApps, getGithubAppOwner(url))
		}
	}
	return githubApps
}

func isGithubAppUrl(url string) bool {
	if strings.HasPrefix(url, "https://github.com") {
		log.Logger().Debugf("url %q is a github app url")
		return len(url) > len("https://github.com")
	}
	return false
}

func getGithubAppOwner(url string) string {
	owner := url[len("https://github.com"):]
	log.Logger().Debugf("github app owner is %q")
	return owner
}
