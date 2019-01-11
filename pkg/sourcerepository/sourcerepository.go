package sourcerepository

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SourceRepositoryService struct {
	client    versioned.Interface
	namespace string
}

func NewSourceRepositoryService(client versioned.Interface, namespace string) SourceRepoer {
	return &SourceRepositoryService{
		client:    client,
		namespace: namespace,
	}
}

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

func (service *SourceRepositoryService) GetSourceRepository(name string) (v1.SourceRepository, error) {
	panic("implement me")
}

func (service *SourceRepositoryService) DeleteSourceRepository(name string) error {
	panic("implement me")
}
