package kube

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetOrCreateRelease creates or updates the given release resource
func GetOrCreateRelease(jxClient versioned.Interface, ns string, release *v1.Release) (*v1.Release, error) {
	releaseInterface := jxClient.JenkinsV1().Releases(ns)
	name := release.Name
	old, err := releaseInterface.Get(name, metav1.GetOptions{})
	if err == nil {
		old.Spec = release.Spec
		answer, err := releaseInterface.Update(old)
		if err != nil {
			return answer, errors.Wrapf(err, "Failed to update Release %s in namespace %s", name, ns)
		}
		return answer, nil
	}
	answer, err := releaseInterface.Create(release)
	if err != nil {
		return answer, errors.Wrapf(err, "Failed to create Release %s in namespace %s", name, ns)
	}
	return answer, nil
}
