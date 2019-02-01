package kube

import "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

// SourceRepoer is responsible for storing information about Source Repositories (aka Applications, Projects)
// go:generate pegomock generate github.com/jenkins-x/jx/pkg/kube SourceRepoer -o mocks/sourcerepoer.go --generate-matchers
type SourceRepoer interface {
	// CreateOrUpdateSourceRepository creates or updates a source repository
	CreateOrUpdateSourceRepository(name, organisation, providerUrl string) error

	// CreateSourceRepository creates an source repository. If a source repository already exists, it will return an error
	CreateSourceRepository(name, organisation, providerUrl string) error

	// DeleteSourceRepository deletes a source repository
	DeleteSourceRepository(name string) error

	// GetSourceRepository gets an source repository, if it exists and returns an error otherwise
	GetSourceRepository(name string) (*v1.SourceRepository, error)

	// ListSourceRepositories gets a list of all the source repositorys
	ListSourceRepositories() (*v1.SourceRepositoryList, error)
}
