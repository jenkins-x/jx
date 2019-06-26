package issues

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/util"
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

func (i *GitIssueProvider) SearchIssuesClosedSince(t time.Time) ([]*gits.GitIssue, error) {
	return i.GitProvider.SearchIssuesClosedSince(i.Owner, i.Repository, t)
}

func (i *GitIssueProvider) IssueURL(key string) string {
	n, err := issueKeyToNumber(key)
	if err != nil {
		return ""
	}
	return i.GitProvider.IssueURL(i.Owner, i.Repository, n, false)
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

func (i *GitIssueProvider) HomeURL() string {
	server := i.GitProvider.Server()
	return util.UrlJoin(server.URL, i.Owner, i.Repository)
}
