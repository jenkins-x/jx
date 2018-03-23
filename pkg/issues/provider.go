package issues

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
)

type IssueProvider interface {
	GetIssue(key string) (*gits.GitIssue, error)

	SearchIssues(query string) ([]*gits.GitIssue, error)

	CreateIssue(issue *gits.GitIssue) (*gits.GitIssue, error)

	CreateIssueComment(key string, comment string) error
}

func CreateIssueProvider(kind string, server *auth.AuthServer, userAuth *auth.UserAuth, project string) (IssueProvider, error) {
	switch kind {
	case Jira:
		return CreateJiraIssueProvider(server, userAuth, project)
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
