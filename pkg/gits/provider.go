package gits

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"strconv"

	"github.com/jenkins-x/jx/pkg/auth"
	"gopkg.in/AlecAivazis/survey.v1"
)

type GitProvider interface {
	ListOrganisations() ([]GitOrganisation, error)

	ListRepositories(org string) ([]*GitRepository, error)

	CreateRepository(org string, name string, private bool) (*GitRepository, error)

	GetRepository(org string, name string) (*GitRepository, error)

	DeleteRepository(org string, name string) error

	ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error)

	RenameRepository(org string, name string, newName string) (*GitRepository, error)

	ValidateRepositoryName(org string, name string) error

	CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error)

	UpdatePullRequestStatus(pr *GitPullRequest) error

	PullRequestLastCommitStatus(pr *GitPullRequest) (string, error)

	ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error)

	MergePullRequest(pr *GitPullRequest, message string) error

	CreateWebHook(data *GitWebHookArguments) error

	IsGitHub() bool

	IsGitea() bool

	GetIssue(org string, name string, number int) (*GitIssue, error)

	CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error)

	HasIssues() bool

	AddPRComment(pr *GitPullRequest, comment string) error

	CreateIssueComment(owner string, repo string, number int, comment string) error

	UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error

	// returns the path relative to the Jenkins URL to trigger webhooks on this kind of repository
	//

	// e.g. for GitHub its /github-webhook/
	// other examples include:
	//
	// * gitlab: /gitlab/notify_commit
	// https://github.com/elvanja/jenkins-gitlab-hook-plugin#notify-commit-hook
	//
	// * git plugin
	// /git/notifyCommit?url=
	// http://kohsuke.org/2011/12/01/polling-must-die-triggering-jenkins-builds-from-a-git-hook/
	//
	// * gitea
	// /gitea-webhook/post
	//
	// * generic webhook
	// /generic-webhook-trigger/invoke?token=abc123
	// https://wiki.jenkins.io/display/JENKINS/Generic+Webhook+Trigger+Plugin

	JenkinsWebHookPath(gitURL string, secret string) string

	Label() string
}

type GitOrganisation struct {
	Login string
}

type GitRepository struct {
	Name             string
	AllowMergeCommit bool
	HTMLURL          string
	CloneURL         string
	SSHURL           string
	Language         string
	Fork             bool
}

type GitPullRequest struct {
	URL            string
	Owner          string
	Repo           string
	Number         *int
	Mergeable      *bool
	Merged         *bool
	State          *string
	StatusesURL    *string
	IssueURL       *string
	DiffURL        *string
	MergeCommitSHA *string
	ClosedAt       *time.Time
	MergedAt       *time.Time
	LastCommitSha  string
}

type GitIssue struct {
	URL           string
	Owner         string
	Repo          string
	Number        *int
	Title         string
	Body          string
	State         *string
	Labels        []GitLabel
	StatusesURL   *string
	IssueURL      *string
	ClosedAt      *time.Time
	IsPullRequest bool
	User          *GitUser
	ClosedBy      *GitUser
	Assignees     []GitUser
}

type GitUser struct {
	URL       string
	Login     string
	Name      string
	Email     string
	AvatarURL string
}

type GitRelease struct {
	Name    string
	TagName string
	Body    string
	URL     string
	HTMLURL string
}

type GitLabel struct {
	URL   string
	Name  string
	Color string
}

type GitRepoStatus struct {
	ID      int64
	Context string
	URL     string

	// State is the current state of the repository. Possible values are:
	// pending, success, error, or failure.
	State string `json:"state,omitempty"`

	// TargetURL is the URL of the page representing this status
	TargetURL string `json:"target_url,omitempty"`

	// Description is a short high level summary of the status.
	Description string
}

type GitPullRequestArguments struct {
	Owner string
	Repo  string
	Title string
	Body  string
	Head  string
	Base  string
}

type GitWebHookArguments struct {
	Owner  string
	Repo   string
	URL    string
	Secret string
}

// IsClosed returns true if the PullRequest has been closed
func (pr *GitPullRequest) IsClosed() bool {
	return pr.ClosedAt != nil
}

// Name returns the textual name of the issue
func (i *GitIssue) Name() string {
	n := i.Number
	if n != nil {
		return "#" + strconv.Itoa(*n)
	}
	return "N/A"
}

func CreateProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	switch server.Kind {
	case "gitea":
		return NewGiteaProvider(server, user)
	default:
		return NewGitHubProvider(server, user)
	}
}

func ProviderAccessTokenURL(kind string, url string) string {
	switch kind {
	case "gitea":
		return GiteaAccessTokenURL(url)
	default:
		return GitHubAccessTokenURL(url)
	}
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

func PickRepositories(provider GitProvider, owner string, message string, selectAll bool, filter string) ([]*GitRepository, error) {
	answer := []*GitRepository{}
	repos, err := provider.ListRepositories(owner)
	if err != nil {
		return answer, err
	}

	repoMap := map[string]*GitRepository{}
	allRepoNames := []string{}
	for _, repo := range repos {
		n := repo.Name
		if n != "" && (filter == "" || strings.Contains(n, filter)) {
			allRepoNames = append(allRepoNames, n)
			repoMap[n] = repo
		}
	}
	if len(allRepoNames) == 0 {
		return answer, fmt.Errorf("No matching repositories could be found!")
	}
	sort.Strings(allRepoNames)

	prompt := &survey.MultiSelect{
		Message: message,
		Options: allRepoNames,
	}
	if selectAll {
		prompt.Default = allRepoNames
	}
	repoNames := []string{}
	err = survey.AskOne(prompt, &repoNames, nil)

	for _, n := range repoNames {
		repo := repoMap[n]
		if repo != nil {
			answer = append(answer, repo)
		}
	}
	return answer, err
}

// IsGitRepoStatusSuccess returns true if all the statuses are successful
func IsGitRepoStatusSuccess(statuses ...*GitRepoStatus) bool {
	for _, status := range statuses {
		if !status.IsSuccess() {
			return false
		}
	}
	return true
}

// IsGitRepoStatusFailed returns true if any of the statuses have failed
func IsGitRepoStatusFailed(statuses ...*GitRepoStatus) bool {
	for _, status := range statuses {
		if status.IsFailed() {
			return true
		}
	}
	return false
}

func (s *GitRepoStatus) IsSuccess() bool {
	return s.State == "success"
}

func (s *GitRepoStatus) IsFailed() bool {
	return s.State == "error" || s.State == "failure"
}

func (i *GitRepositoryInfo) PickOrCreateProvider(authConfigSvc auth.AuthConfigService, message string, batchMode bool, gitKind string) (GitProvider, error) {
	config := authConfigSvc.Config()
	hostUrl := i.HostURLWithoutUser()
	server := config.GetOrCreateServer(hostUrl)
	userAuth, err := config.PickServerUserAuth(server, message, batchMode)
	if err != nil {
		return nil, err
	}
	return i.CreateProviderForUser(server, userAuth, gitKind)
}

func (i *GitRepositoryInfo) CreateProviderForUser(server *auth.AuthServer, user *auth.UserAuth, gitKind string) (GitProvider, error) {
	if i.Host == GitHubHost {
		return NewGitHubProvider(server, user)
	}
	if gitKind != "" && server.Kind != gitKind {
		server.Kind = gitKind
	}
	return CreateProvider(server, user)
}

func (i *GitRepositoryInfo) CreateProvider(authConfigSvc auth.AuthConfigService, gitKind string) (GitProvider, error) {
	config := authConfigSvc.Config()
	hostUrl := i.HostURLWithoutUser()
	server := config.GetOrCreateServer(hostUrl)
	url := server.URL
	if gitKind != "" {
		server.Kind = gitKind
	}
	userAuths := authConfigSvc.Config().FindUserAuths(url)
	if len(userAuths) == 0 {
		kind := server.Kind
		if kind != "" {
			userAuth := auth.CreateAuthUserFromEnvironment(strings.ToUpper(kind))
			if !userAuth.IsInvalid() {
				return CreateProvider(server, &userAuth)
			}
		}
		userAuth := auth.CreateAuthUserFromEnvironment("GIT")
		if !userAuth.IsInvalid() {
			return CreateProvider(server, &userAuth)
		}
	}
	if len(userAuths) > 0 {
		// TODO use default user???
		auth := userAuths[0]
		return CreateProvider(server, auth)
	}
	return nil, fmt.Errorf("Could not create Git provider for host %s as no user auths could be found", hostUrl)
}
