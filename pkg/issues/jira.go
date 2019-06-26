package issues

import (
	"bytes"
	"fmt"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

type JiraService struct {
	jiraClient *jira.Client
	server     auth.Server
	project    string
	git        gits.Gitter
}

func CreateJiraIssueProvider(server auth.Server, project string, batchMode bool, git gits.Gitter) (IssueProvider, error) {
	u := server.URL
	if u == "" {
		return nil, fmt.Errorf("No base URL for server!")
	}
	user, err := server.GetCurrentUser()
	if err != nil {
		return nil, err
	}
	if user.IsInvalid() {
		log.Logger().Warnf("No valid authentication found for JIRA server %s so using anonymous access", u)
	}

	tp := jira.BasicAuthTransport{
		Username: user.Username,
		Password: user.ApiToken,
	}
	httpClient := tp.Client()
	jiraClient, _ := jira.NewClient(httpClient, u)
	return &JiraService{
		jiraClient: jiraClient,
		server:     server,
		project:    project,
		git:        git,
	}, nil
}

func (i *JiraService) GetIssue(key string) (*gits.GitIssue, error) {
	issue, _, err := i.jiraClient.Issue.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return i.jiraToGitIssue(issue), nil
}

func (i *JiraService) SearchIssues(query string) ([]*gits.GitIssue, error) {
	jql := "project = " + i.project + " AND status NOT IN (Closed, Resolved)"
	if query != "" {
		jql += " AND text ~ " + query
	}
	answer := []*gits.GitIssue{}
	issues, _, err := i.jiraClient.Issue.Search(jql, nil)
	if err != nil {
		return answer, err
	}
	for _, issue := range issues {
		answer = append(answer, i.jiraToGitIssue(&issue))
	}
	return answer, nil
}

func (i *JiraService) SearchIssuesClosedSince(t time.Time) ([]*gits.GitIssue, error) {
	log.Logger().Warn("TODO SearchIssuesClosedSince() not yet implemented for JIRA")
	return nil, nil
}

func (i *JiraService) CreateIssue(issue *gits.GitIssue) (*gits.GitIssue, error) {
	project, _, err := i.jiraClient.Project.Get(i.project)
	if err != nil {
		return nil, fmt.Errorf("Could not find project %s: %s", i.project, err)
	}
	ji := i.gitToJiraIssue(issue)
	issueTypes := project.IssueTypes
	if len(issueTypes) > 0 {
		it := issueTypes[0]
		ji.Fields.Type.Name = it.Name
	}
	jira, resp, err := i.jiraClient.Issue.Create(ji)
	if err != nil {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(resp.Body); err != nil {
			return nil, err
		}
		msg := buf.String()
		return nil, fmt.Errorf("Failed to create issue: %s due to: %s", msg, err)
	}
	return i.jiraToGitIssue(jira), nil
}

func (i *JiraService) CreateIssueComment(key string, comment string) error {
	return fmt.Errorf("TODO")
}

func (i *JiraService) IssueURL(key string) string {
	return util.UrlJoin(i.server.URL, "browse", key)
}

func (i *JiraService) jiraToGitIssue(issue *jira.Issue) *gits.GitIssue {
	answer := &gits.GitIssue{}
	key := issue.Key
	answer.Key = key
	answer.URL = i.IssueURL(key)
	fields := issue.Fields
	if fields != nil {
		answer.Title = fields.Summary
		answer.Body = fields.Description
		answer.Labels = gits.ToGitLabels(fields.Labels)
		answer.ClosedAt = jiraTimeToTimeP(fields.Resolutiondate)
		answer.User = jiraUserToGitUser(fields.Reporter)
		assignee := jiraUserToGitUser(fields.Assignee)
		if assignee != nil {
			answer.Assignees = []gits.GitUser{*assignee}
		}
	}
	return answer
}

func jiraUserToGitUser(user *jira.User) *gits.GitUser {
	if user == nil {
		return nil
	}
	return &gits.GitUser{
		AvatarURL: jiraAvatarUrl(user),
		Name:      user.Name,
		Login:     user.Key,
		Email:     user.EmailAddress,
	}
}
func jiraAvatarUrl(user *jira.User) string {
	answer := ""
	if user != nil {
		av := user.AvatarUrls
		answer = av.Four8X48
		if answer == "" {
			answer = av.Three2X32
		}
		if answer == "" {
			answer = av.Two4X24
		}
		if answer == "" {
			answer = av.One6X16
		}
	}
	return answer
}

func jiraTimeToTimeP(t jira.Time) *time.Time {
	tt := time.Time(t)
	return &tt
}

func (i *JiraService) gitToJiraIssue(issue *gits.GitIssue) *jira.Issue {
	answer := &jira.Issue{
		Fields: &jira.IssueFields{
			Project: jira.Project{
				Key: i.project,
			},
			Summary:     issue.Title,
			Description: issue.Body,
			Type: jira.IssueType{
				Name: "Bug",
			},
		},
	}
	return answer
}

func (i *JiraService) ServerName() string {
	return i.server.URL
}

func (i *JiraService) HomeURL() string {
	return util.UrlJoin(i.server.URL, "browse", i.project)
}
