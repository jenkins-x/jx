package issues

import (
	"fmt"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
)

type IssueProvider interface {
	// GetIssue returns the issue of the given key
	GetIssue(key string) (*gits.GitIssue, error)

	// SearchIssues searches for issues (open by default)
	SearchIssues(query string) ([]*gits.GitIssue, error)

	// SearchIssuesClosedSince searches the issues closed since the given da
	SearchIssuesClosedSince(t time.Time) ([]*gits.GitIssue, error)

	// Creates a new issue in the current project
	CreateIssue(issue *gits.GitIssue) (*gits.GitIssue, error)

	// Creates a comment on the given issue
	CreateIssueComment(key string, comment string) error

	// IssueURL returns the URL of the given issue for this project
	IssueURL(key string) string

	// HomeURL returns the home URL of the issue tracker
	HomeURL() string
}

func CreateIssueProvider(kind string, server *auth.ServerAuth, userAuth *auth.UserAuth, project string, batchMode bool, git gits.Gitter) (IssueProvider, error) {
	switch kind {
	case Jira:
		return CreateJiraIssueProvider(server, userAuth, project, batchMode, git)
	default:
		return nil, fmt.Errorf("Unsupported issue provider kind: %s", kind)
	}
}

func ProviderAccessTokenURL(kind string, url string) string {
	switch kind {
	case Jira:
		// TODO handle on premise servers too by detecting the URL is at atlassian.com
		return "https://id.atlassian.com/manage/api-tokens"
	default:
		return ""
	}
}

// GetIssueProvider returns the kind of issue provider
func GetIssueProvider(tracker IssueProvider) string {
	_, ok := tracker.(*JiraService)
	if ok {
		return Jira
	}
	return Git
}
