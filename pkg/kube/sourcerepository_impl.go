package kube

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

//SourceRepositoryService is the implementation of SourceRepoer
type SourceRepositoryService struct {
	client    versioned.Interface
	namespace string
}

// NewSourceRepositoryService creates a new Service for interacting with the Source Repository Custom Resource
func NewSourceRepositoryService(client versioned.Interface, namespace string) SourceRepoer {
	return &SourceRepositoryService{
		client:    client,
		namespace: namespace,
	}
}

// CreateOrUpdateSourceRepository creates a repo if it doesn't exist or updates it if the URL has changed
func (service *SourceRepositoryService) CreateOrUpdateSourceRepository(name, organisation, providerURL string) error {
	//FIXME: repo is not always == name, need to find a better value for ObjectMeta.Name!

	// for now lets convert to a safe name using the organisation + repo name
	resourceName := ToValidName(organisation + "-" + name)

	repositories := service.client.JenkinsV1().SourceRepositories(service.namespace)
	description := fmt.Sprintf("Imported application for %s/%s", organisation, name)

	_, err := repositories.Create(&v1.SourceRepository{
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
			return errors.Wrapf(err, "failed to create SourceRepository %s and cannot get it either: %s", resourceName, err2.Error())
		}
		copy := *sr
		copy.Spec.Description = description
		copy.Spec.Org = organisation
		copy.Spec.Provider = providerURL
		copy.Spec.Repo = name
		if reflect.DeepEqual(&copy.Spec, sr.Spec) {
			return nil
		}
		_, err = repositories.Update(&copy)
		if err != nil {
			return errors.Wrapf(err, "failed to update SourceRepository %s", resourceName)
		}
	}
	return nil
}

// CreateSourceRepository creates a repo. If a repo already exists, it will return an error
func (service *SourceRepositoryService) CreateSourceRepository(name, organisation, providerURL string) error {
	//FIXME: repo is not always == name, need to find a better value for ObjectMeta.Name!
	// for now lets convert to a safe name using the organisation + repo name
	resourceName := ToValidName(organisation + "-" + name)

	_, err := service.client.JenkinsV1().SourceRepositories(service.namespace).Create(&v1.SourceRepository{
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

// GetSourceRepository gets repo, if it exists and returns an error otherwise
func (service *SourceRepositoryService) GetSourceRepository(name string) (*v1.SourceRepository, error) {
	return service.client.JenkinsV1().SourceRepositories(service.namespace).Get(name, metav1.GetOptions{})
}

// DeleteSourceRepository deletes a repo
func (service *SourceRepositoryService) DeleteSourceRepository(name string) error {
	return service.client.JenkinsV1().SourceRepositories(service.namespace).Delete(name, &metav1.DeleteOptions{})
}

// ListSourceRepositories gets a list of all the repos
func (service *SourceRepositoryService) ListSourceRepositories() (*v1.SourceRepositoryList, error) {
	return service.client.JenkinsV1().SourceRepositories(service.namespace).List(metav1.ListOptions{})
}
