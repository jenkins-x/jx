package gits

import (
	"fmt"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
)

type GiteaProvider struct {
	Username string
	Client   *gitea.Client

	Server auth.AuthServer
	User   auth.UserAuth
}

func NewGiteaProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	client := gitea.NewClient(server.URL, user.ApiToken)

	provider := GiteaProvider{
		Client:   client,
		Server:   *server,
		User:     *user,
		Username: user.Username,
	}

	return &provider, nil
}

func (p *GiteaProvider) ListOrganisations() ([]GitOrganisation, error) {
	answer := []GitOrganisation{}
	orgs, err := p.Client.ListMyOrgs()
	if err != nil {
		return answer, err
	}

	for _, org := range orgs {
		name := org.UserName
		if name != "" {
			o := GitOrganisation{
				Login: name,
			}
			answer = append(answer, o)
		}
	}
	return answer, nil
}

func (p *GiteaProvider) ListRepositories(org string) ([]*GitRepository, error) {
	answer := []*GitRepository{}
	if org == "" {
		repos, err := p.Client.ListMyRepos()
		if err != nil {
			return answer, err
		}
		for _, repo := range repos {
			answer = append(answer, toGiteaRepo(repo.Name, repo))
		}
		return answer, nil
	}
	repos, err := p.Client.ListOrgRepos(org)
	if err != nil {
		return answer, err
	}
	for _, repo := range repos {
		answer = append(answer, toGiteaRepo(repo.Name, repo))
	}
	return answer, nil
}

func (p *GiteaProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	options := gitea.CreateRepoOption{
		Name:    name,
		Private: private,
	}
	repo, err := p.Client.CreateRepo(options)
	if err != nil {
		return nil, fmt.Errorf("Failed to create repository %s/%s due to: %s", org, name, err)
	}
	return toGiteaRepo(name, repo), nil
}

func (p *GiteaProvider) DeleteRepository(org string, name string) error {
	owner := org
	if owner == "" {
		owner = p.Username
	}
	err := p.Client.DeleteRepo(owner, name)
	if err != nil {
		return fmt.Errorf("Failed to delete repository %s/%s due to: %s", owner, name, err)
	}
	return err
}

func toGiteaRepo(name string, repo *gitea.Repository) *GitRepository {
	return &GitRepository{
		Name:             name,
		AllowMergeCommit: true,
		CloneURL:         repo.CloneURL,
		HTMLURL:          repo.HTMLURL,
		SSHURL:           repo.SSHURL,
	}
}

func (p *GiteaProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	repoConfig := gitea.CreateForkOption{
		Organization: &destinationOrg,
	}
	repo, err := p.Client.CreateFork(originalOrg, name, repoConfig)
	if err != nil {
		msg := ""
		if destinationOrg != "" {
			msg = fmt.Sprintf(" to %s", destinationOrg)
		}
		owner := destinationOrg
		if owner == "" {
			owner = p.Username
		}
		if strings.Contains(err.Error(), "try again later") {
			fmt.Printf("Waiting for the fork of %s/%s to appear...\n", owner, name)
			// lets wait for the fork to occur...
			start := time.Now()
			deadline := start.Add(time.Minute)
			for {
				time.Sleep(5 * time.Second)
				repo, err = p.Client.GetRepo(owner, name)
				if repo != nil && err == nil {
					break
				}
				t := time.Now()
				if t.After(deadline) {
					return nil, fmt.Errorf("Gave up waiting for Repository %s/%s to appear: %s", owner, name, err)
				}
			}
		} else {
			return nil, fmt.Errorf("Failed to fork repository %s/%s%s due to: %s", originalOrg, name, msg, err)
		}
	}
	return toGiteaRepo(name, repo), nil
}

func (p *GiteaProvider) CreateWebHook(data *GitWebHookArguments) error {
	owner := data.Owner
	if owner == "" {
		owner = p.Username
	}
	repo := data.Repo
	if repo == "" {
		return fmt.Errorf("Missing property Repo")
	}
	webhookUrl := data.URL
	if repo == "" {
		return fmt.Errorf("Missing property URL")
	}
	hooks, err := p.Client.ListRepoHooks(owner, repo)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		s := hook.Config["url"]
		if s == webhookUrl {
			fmt.Printf("Already has a webhook registered for %s\n", webhookUrl)
			return nil
		}
	}
	config := map[string]string{
		"url":          webhookUrl,
		"content_type": "json",
	}
	if data.Secret != "" {
		config["secret"] = data.Secret
	}
	hook := gitea.CreateHookOption{
		Config: config,
		Events: []string{"*"},
	}
	fmt.Printf("Creating github webhook for %s/%s for url %s\n", owner, repo, webhookUrl)
	_, err = p.Client.CreateRepoHook(owner, repo, hook)
	return err
}

func (p *GiteaProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	owner := data.Owner
	repo := data.Repo
	title := data.Title
	body := data.Body
	head := data.Head
	base := data.Base
	config := gitea.CreatePullRequestOption{}
	if title != "" {
		config.Title = title
	}
	if body != "" {
		config.Body = body
	}
	if head != "" {
		config.Head = head
	}
	if base != "" {
		config.Base = base
	}
	pr, err := p.Client.CreatePullRequest(owner, repo, config)
	if err != nil {
		return nil, err
	}
	return &GitPullRequest{
		URL: pr.HTMLURL,
	}, nil
}

func (p *GiteaProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	return nil, fmt.Errorf("Rename of repositories is not supported for gitea")
}

func (p *GiteaProvider) ValidateRepositoryName(org string, name string) error {
	_, err := p.Client.GetRepo(org, name)
	if err == nil {
		return fmt.Errorf("Repository %s already exists", GitRepoName(org, name))
	}
	if strings.Contains(err.Error(), "404") {
		return nil
	}
	return err
}

func (p *GiteaProvider) IsGitHub() bool {
	return false
}

func (p *GiteaProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/generic-webhook-trigger/invoke"
}

func GiteaAccessTokenURL(url string) string {
	return util.UrlJoin(url, "/user/settings/applications")
}

func (p *GiteaProvider) Label() string {
	return p.Server.Label()
}
