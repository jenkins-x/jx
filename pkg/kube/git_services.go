package kube

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureGitServiceExistsForHost ensures that there is a GitService CRD for the given host and kind
func EnsureGitServiceExistsForHost(jxClient versioned.Interface, devNs string, kind string, name string, gitUrl string, out io.Writer) error {
	if kind == "" || kind == "github" || gitUrl == "" {
		return nil
	}

	gitServices := jxClient.JenkinsV1().GitServices(devNs)
	list, err := gitServices.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, gs := range list.Items {
		if gs.Spec.URL == gitUrl {
			oldKind := gs.Spec.GitKind
			if oldKind != kind {
				fmt.Fprintf(out, "Updating GitService %s as the kind has changed from %s to %s\n", gs.Name, oldKind, kind)
				gs.Spec.GitKind = kind
				_, err = gitServices.Update(&gs)
				if err != nil {
					return fmt.Errorf("Failed to update kind on GitService with name %s: %s", gs.Name, err)
				}
				return err
			} else {
				return nil
			}
		}
	}
	if name == "" {
		u, err := url.Parse(gitUrl)
		if err != nil {
			return fmt.Errorf("No name supplied and could not parse URL %s due to %s", u, err)
		}
		name = u.Host
	}

	// not found so lets create a new GitService
	gitSvc := &v1.GitService{
		ObjectMeta: metav1.ObjectMeta{
			Name: ToValidNameWithDots(name),
		},
		Spec: v1.GitServiceSpec{
			Name:    name,
			URL:     gitUrl,
			GitKind: kind,
		},
	}
	current, err := gitServices.Get(name, metav1.GetOptions{})
	if err != nil {
		_, err = gitServices.Create(gitSvc)
		if err != nil {
			return fmt.Errorf("Failed to create GitService with name %s: %s", gitSvc.Name, err)
		}
	} else if current != nil {
		if current.Spec.URL != gitSvc.Spec.URL || current.Spec.GitKind != gitSvc.Spec.GitKind {
			current.Spec.URL = gitSvc.Spec.URL
			current.Spec.GitKind = gitSvc.Spec.GitKind

			_, err = gitServices.Update(current)
			if err != nil {
				return fmt.Errorf("Failed to update GitService with name %s: %s", gitSvc.Name, err)
			}
		}
	}
	return nil
}

// GetGitServiceKind returns the kind of the given host if one can be found or ""
func GetGitServiceKind(jxClient versioned.Interface, kubeClient kubernetes.Interface, devNs string, gitServiceUrl string) (string, error) {
	answer := gits.SaasGitKind(gitServiceUrl)
	if answer != "" {
		return answer, nil
	}
	cm, err := kubeClient.CoreV1().ConfigMaps(devNs).Get(ConfigMapJenkinsXGitKinds, metav1.GetOptions{})
	if err == nil {
		answer = GetGitServiceKindFromConfigMap(cm, gitServiceUrl)
		if answer != "" {
			return answer, nil
		}
	}

	gitServices := jxClient.JenkinsV1().GitServices(devNs)
	list, err := gitServices.List(metav1.ListOptions{})
	if err == nil {
		for _, gs := range list.Items {
			if gs.Spec.URL == gitServiceUrl {
				return gs.Spec.GitKind, nil
			}
		}
	}
	// TODO should we default to github?
	return answer, nil
}

func GetGitServiceKindFromConfigMap(cm *corev1.ConfigMap, gitServiceUrl string) string {
	gitServiceUrl = strings.TrimSuffix(gitServiceUrl, "/")
	for k, v := range cm.Data {
		if strings.TrimSpace(v) != "" {
			m := map[string]string{}
			err := yaml.Unmarshal([]byte(v), &m)
			if err != nil {
				fmt.Printf("Warning could not parse %s YAML %s: %s due to: %s", ConfigMapJenkinsXGitKinds, k, v, err)
			} else {
				for _, u := range m {
					if u == gitServiceUrl {
						return k
					}
					if k == "github" {
						// lets trim /api/ paths from github repos
						idx := strings.LastIndex(u, "/api/")
						if idx > 0 {
							u2 := u[0:idx]
							if gitServiceUrl == u2 {
								return k
							}
						}
					}
				}
			}
		}
	}
	return ""
}
