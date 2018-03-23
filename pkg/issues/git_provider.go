package issues

import (
	"fmt"
	"strconv"

	"github.com/jenkins-x/jx/pkg/gits"
)

type GitIssueProvider struct {
	GitProvider gits.GitProvider
	Owner       string
	Repository  string
}

func CreateGitIssueProvider(gitProvider gits.GitProvider, owner string, repository string) (IssueProvider, error) {
	if owner == "" {
		return nil, fmt.Errorf("No owner specified")
	}
	if repository == "" {
		return nil, fmt.Errorf("No owner specified")
	}
	return &GitIssueProvider{
		GitProvider: gitProvider,
		Owner:       owner,
		Repository:  repository,
	}, nil
}

func (i *GitIssueProvider) GetIssue(key string) (*gits.GitIssue, error) {
	n, err := issueKeyToNumber(key)
	if err != nil {
		return nil, err
	}
	return i.GitProvider.GetIssue(i.Owner, i.Repository, n)
}

func (i *GitIssueProvider) SearchIssues(query string) ([]*gits.GitIssue, error) {
	return i.GitProvider.SearchIssues(i.Owner, i.Repository, query)
}

func issueKeyToNumber(key string) (int, error) {
	n, err := strconv.Atoi(key)
	if err != nil {
		return n, fmt.Errorf("Failed to convert issue key '%s' to number: %s", key, err)
	}
	return n, nil
}

func (i *GitIssueProvider) CreateIssue(issue *gits.GitIssue) (*gits.GitIssue, error) {
	return i.GitProvider.CreateIssue(i.Owner, i.Repository, issue)
}

func (i *GitIssueProvider) CreateIssueComment(key string, comment string) error {
	n, err := issueKeyToNumber(key)
	if err != nil {
		return err
	}
	return i.GitProvider.CreateIssueComment(i.Owner, i.Repository, n, comment)
}
