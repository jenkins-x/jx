package gits

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/auth"
	"gopkg.in/AlecAivazis/survey.v1"
	"sort"
)

type GitProvider interface {
	ListOrganisations() ([]GitOrganisation, error)

	CreateRepository(org string, name string, private bool) (*GitRepository, error)

	ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error)

	RenameRepository(org string, name string, newName string) (*GitRepository, error)

	ValidateRepositoryName(org string, name string) error

	CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error)

	IsGitHub() bool
}

type GitOrganisation struct {
	Login string
}

type GitRepository struct {
	AllowMergeCommit bool
	HTMLURL          string
	CloneURL         string
	SSHURL           string
}

type GitPullRequest struct {
	URL string
}

type GitPullRequestArguments struct {
	Owner string
	Repo  string
	Title string
	Body  string
	Head  string
	Base  string
}

func CreateProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	// TODO lets default to github
	return NewGitHubProvider(server, user)
}

// PickOrganisation picks an organisations login if there is one available
func PickOrganisation(provider GitProvider, userName string) (string, error) {
	answer := ""
	orgs, err := provider.ListOrganisations()
	if err != nil {
		return answer, err
	}
	if len(orgs) == 0 {
		return answer, nil
	}
	if len(orgs) == 1 {
		return orgs[0].Login, nil
	}
	orgNames := []string{userName}
	for _, o := range orgs {
		name := o.Login
		if name != "" {
			orgNames = append(orgNames, name)
		}
	}
	sort.Strings(orgNames)
	orgName := ""
	prompt := &survey.Select{
		Message: "Which organisation do you want to use?",
		Options: orgNames,
		Default: userName,
	}
	err = survey.AskOne(prompt, &orgName, nil)
	if err != nil {
		return "", err
	}
	if orgName == userName {
		return "", nil
	}
	return orgName, nil
}

func (i *GitRepositoryInfo) PickOrCreateProvider(authConfigSvc auth.AuthConfigService) (GitProvider, error) {
	config := authConfigSvc.Config()
	server := config.GetOrCreateServer(i.Host)
	userAuth, err := config.PickServerUserAuth(server, "git user name")
	if err != nil {
		return nil, err
	}
	return i.CreateProviderForUser(server, &userAuth)
}

func (i *GitRepositoryInfo) CreateProviderForUser(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	if i.Host == GitHubHost {
		return NewGitHubProvider(server, user)
	}
	return nil, fmt.Errorf("Git provider not supported for host %s", i.Host)
}
