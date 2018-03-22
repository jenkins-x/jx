package issues

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
)

type JiraService struct {
	JiraClient *jira.Client
	Server     *auth.AuthServer
	Project    string
}

func CreateJiraIssueProvider(server *auth.AuthServer, userAuth *auth.UserAuth, project string) (IssueProvider, error) {
	if server.URL == "" {
		return nil, fmt.Errorf("No base URL for server!")
	}
	var httpClient *http.Client
	if userAuth != nil && !userAuth.IsInvalid() {
		user := userAuth.Username
		tp := jira.BasicAuthTransport{
			Username: user,
			Password: userAuth.ApiToken,
		}
		/*
		 */
		httpClient = tp.Client()
	}
	fmt.Printf("Conecting to server %s with user %s token %s\n", server.URL, userAuth.Username, userAuth.ApiToken)
	jiraClient, _ := jira.NewClient(httpClient, server.URL)
	return &JiraService{
		JiraClient: jiraClient,
		Server:     server,
		Project:    project,
	}, nil
}

func (i *JiraService) GetIssue(key string) (*gits.GitIssue, error) {
	issue, _, err := i.JiraClient.Issue.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return jiraToGitIssue(issue), nil
}

func (i *JiraService) SearchIssues(query string) ([]*gits.GitIssue, error) {
	jql := "project = " + i.Project + " AND status NOT IN (Closed, Resolved)"
	if query != "" {
		jql += " AND text ~ " + query
	}
	answer := []*gits.GitIssue{}
	issues, _, err := i.JiraClient.Issue.Search(jql, nil)
	if err != nil {
		return answer, err
	}
	for _, issue := range issues {
		answer = append(answer, jiraToGitIssue(&issue))
	}
	return answer, nil
}

func (i *JiraService) CreateIssue(issue *gits.GitIssue) (*gits.GitIssue, error) {
	if !i.JiraClient.Authentication.Authenticated() {
		return nil, fmt.Errorf("Cannot create issue as there is no authentication for server %s", i.ServerName())
	}
	jira, _, err := i.JiraClient.Issue.Create(gitToJiraIssue(issue))
	if err != nil {
		return nil, err
	}
	return jiraToGitIssue(jira), nil
}

func (i *JiraService) CreateIssueComment(key string, comment string) error {
	if !i.JiraClient.Authentication.Authenticated() {
		return fmt.Errorf("Cannot create issue comments as there is no authentication for server %s", i.ServerName())
	}
	return fmt.Errorf("TODO")
}

func jiraToGitIssue(issue *jira.Issue) *gits.GitIssue {
	answer := &gits.GitIssue{}
	fields := issue.Fields
	if fields != nil {
		answer.Title = fields.Summary
		answer.Body = fields.Description
		answer.Labels = gits.ToGitLabels(fields.Labels)
		answer.ClosedAt = jiraTimeToTimeP(fields.Resolutiondate)
	}
	return answer
}

func jiraTimeToTimeP(t jira.Time) *time.Time {
	tt := time.Time(t)
	return &tt
}

func gitToJiraIssue(issue *gits.GitIssue) *jira.Issue {
	answer := &jira.Issue{}
	return answer
}

func (i *JiraService) ServerName() string {
	return i.Server.URL
}
