package kube

import (
	"fmt"
	"reflect"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetOrCreateSourceRepository gets or creates the SourceRepository for the given repository name and organisation
func GetOrCreateSourceRepository(jxClient versioned.Interface, ns string, name, organisation, providerURL string) (*v1.SourceRepository, error) {
	resourceName := ToValidName(organisation + "-" + name)

	repositories := jxClient.JenkinsV1().SourceRepositories(ns)
	description := fmt.Sprintf("Imported application for %s/%s", organisation, name)

	answer, err := repositories.Create(&v1.SourceRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Spec: v1.SourceRepositorySpec{
			Description: description,
			Org:         organisation,
			Provider:    providerURL,
			Repo:        name,
		},
	})
	if err != nil {
		// lets see if it already exists
		sr, err2 := repositories.Get(resourceName, metav1.GetOptions{})
		if err2 != nil {
			return answer, errors.Wrapf(err, "failed to create SourceRepository %s and cannot get it either: %s", resourceName, err2.Error())
		}
		answer = sr
		copy := *sr
		copy.Spec.Description = description
		copy.Spec.Org = organisation
		copy.Spec.Provider = providerURL
		copy.Spec.Repo = name
		if reflect.DeepEqual(&copy.Spec, sr.Spec) {
			return answer, nil
		}
		answer, err = repositories.PatchUpdate(&copy)
		if err != nil {
			return answer, errors.Wrapf(err, "failed to update SourceRepository %s", resourceName)
		}
	}
	return answer, nil
}

// CreateSourceRepository creates a repo. If a repo already exists, it will return an error
func CreateSourceRepository(client versioned.Interface, namespace string, name, organisation, providerURL string) error {
	//FIXME: repo is not always == name, need to find a better value for ObjectMeta.Name!
	// for now lets convert to a safe name using the organisation + repo name
	resourceName := ToValidName(organisation + "-" + name)

	_, err := client.JenkinsV1().SourceRepositories(namespace).Create(&v1.SourceRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Spec: v1.SourceRepositorySpec{
			Description: fmt.Sprintf("Imported application for %s/%s", organisation, name),
			Org:         organisation,
			Provider:    providerURL,
			Repo:        name,
		},
	})
	if err != nil {
		return err
	}

	return nil
}
