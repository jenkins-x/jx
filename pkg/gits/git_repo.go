package gits

import (
	"github.com/jenkins-x/jx/pkg/auth"
)

type CreateRepoData struct {
	Organisation string
	RepoName     string
	FullName     string
	PrivateRepo  bool
	GitProvider  GitProvider
	GitServer    auth.Server
}

type GitRepositoryOptions struct {
	ServerURL  string
	ServerKind string
	Username   string
	ApiToken   string
	Owner      string
	RepoName   string
	Private    bool
}

// GetRepository returns the repository if it already exists
func (d *CreateRepoData) GetRepository() (*GitRepository, error) {
	return d.GitProvider.GetRepository(d.Organisation, d.RepoName)
}

// CreateRepository creates the repository - failing if it already exists
func (d *CreateRepoData) CreateRepository() (*GitRepository, error) {
	return d.GitProvider.CreateRepository(d.Organisation, d.RepoName, d.PrivateRepo)
}
