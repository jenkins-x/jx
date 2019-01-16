package sourcerepository

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

//FIXME: repo is not always == name, need to find a better value for ObjectMeta.Name!
//CreateSourceRepository creates a repo. If a repo already exists, it will return an error
func (service *SourceRepositoryService) CreateSourceRepository(name, organisation, providerUrl string) error {
	_, err := service.client.JenkinsV1().SourceRepositories(service.namespace).Create(&v1.SourceRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.SourceRepositorySpec{
			Description: fmt.Sprintf("Imported application for %s/%s", organisation, name),
			Org:         organisation,
			Provider:    providerUrl,
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
