package kube

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureGitServiceExistsForHost ensures that there is a GitService CRD for the given host and kind
func EnsureGitServiceExistsForHost(jxClient *versioned.Clientset, devNs string, kind string, host string, out io.Writer) error {
	if kind == "" || kind == "github" || host == "" {
		return nil
	}

	gitServices := jxClient.JenkinsV1().GitServices(devNs)
	list, err := gitServices.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, gs := range list.Items {
		if gs.Spec.Host == host {
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

	// not found so lets create a new GitService
	gitSvc := &v1.GitService{
		ObjectMeta: metav1.ObjectMeta{
			Name: ToValidNameWithDots(host),
		},
		Spec: v1.GitServiceSpec{
			Host:    host,
			GitKind: kind,
		},
	}
	_, err = gitServices.Create(gitSvc)
	if err != nil {
		return fmt.Errorf("Failed to create  GitService with name %s: %s", gitSvc.Name, err)
	}
	return nil
}

// GetGitServiceKind returns the kind of the given host if one can be found or ""
func GetGitServiceKind(jxClient *versioned.Clientset, devNs string, host string) (string, error) {
	answer := ""
	gitServices := jxClient.JenkinsV1().GitServices(devNs)
	list, err := gitServices.List(metav1.ListOptions{})
	if err != nil {
		return answer, err
	}
	for _, gs := range list.Items {
		if gs.Spec.Host == host {
			return gs.Spec.GitKind, nil
		}
	}
	return answer, nil
}
