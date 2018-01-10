package gits

import (
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"context"
	"fmt"
)

type GitHubProvider struct {
	Username string
	Client   *github.Client
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
}

func NewGitHubProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	ctx := context.Background()

	provider := GitHubProvider{
		Server:   *server,
		User:     *user,
		Context:  ctx,
		Username: user.Username,
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: user.ApiToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	provider.Client = github.NewClient(tc)
	return &provider, nil
}

func (p *GitHubProvider) ListOrganisations() ([]GitOrganisation, error) {
	answer := []GitOrganisation{}
	orgs, _, err := p.Client.Organizations.List(p.Context, p.Username, nil)
	if err != nil {
		return answer, err
	}

	for _, org := range orgs {
		name := org.Login
		if name != nil {
			o := GitOrganisation{
				Login: *name,
			}
			answer = append(answer, o)
		}
	}
	return answer, nil
}

func (p *GitHubProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	repoConfig := &github.Repository{
		Name:    github.String(name),
		Private: github.Bool(private),
	}
	repo, _, err := p.Client.Repositories.Create(p.Context, org, repoConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create repository %s/%s due to: %s", org, name, err)
	}
	answer := &GitRepository{
		AllowMergeCommit: asBool(repo.AllowMergeCommit),
		CloneURL: asText(repo.CloneURL),
		HTMLURL: asText(repo.HTMLURL),
		SSHURL: asText(repo.SSHURL),
	}
	return answer, nil
}

func (p *GitHubProvider) ValidateRepositoryName(org string, name string) error {
	_, r, err := p.Client.Repositories.Get(p.Context, org, name)
	if err == nil {
		return fmt.Errorf("Repository %s already exists", GitRepoName(org, name))
	}
	if r.StatusCode == 404 {
		return nil
	}
	return err
}


func asBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func asText(text *string) string {
	if text != nil {
		return *text
	}
	return ""
}