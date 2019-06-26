package gits

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
)

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
	Stars            int
	URL              string
	Scheme           string
	Host             string
	Organisation     string
	Project          string
	Private          bool
}

type GitPullRequest struct {
	URL                string
	Author             *GitUser
	Owner              string
	Repo               string
	Number             *int
	Mergeable          *bool
	Merged             *bool
	HeadRef            *string
	State              *string
	StatusesURL        *string
	IssueURL           *string
	DiffURL            *string
	MergeCommitSHA     *string
	ClosedAt           *time.Time
	MergedAt           *time.Time
	LastCommitSha      string
	Title              string
	Body               string
	Assignees          []*GitUser
	RequestedReviewers []*GitUser
	Labels             []*Label
	UpdatedAt          *time.Time
}

// Label represents a label on an Issue
type Label struct {
	ID          *int64
	URL         *string
	Name        *string
	Color       *string
	Description *string
	Default     *bool
}

type GitCommit struct {
	SHA       string
	Message   string
	Author    *GitUser
	URL       string
	Branch    string
	Committer *GitUser
}

type ListCommitsArguments struct {
	SHA     string
	Path    string
	Author  string
	Since   time.Time
	Until   time.Time
	Page    int
	PerPage int
}

type GitIssue struct {
	URL           string
	Owner         string
	Repo          string
	Number        *int
	Key           string
	Title         string
	Body          string
	State         *string
	Labels        []GitLabel
	StatusesURL   *string
	IssueURL      *string
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
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
	Name          string
	TagName       string
	Body          string
	URL           string
	HTMLURL       string
	DownloadCount int
	Assets        *[]GitReleaseAsset
}

// GitReleaseAsset represents a release stored in Git
type GitReleaseAsset struct {
	BrowserDownloadURL string
	Name               string
	ContentType        string
}

type GitLabel struct {
	URL   string
	Name  string
	Color string
}

type GitRepoStatus struct {
	ID      string
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
	Title         string
	Body          string
	Head          string
	Base          string
	GitRepository *GitRepository
	Labels        []string
}

func (a *GitPullRequestArguments) String() string {
	return fmt.Sprintf("Title: %s; Body: %s; Head: %s; Base: %s; Labels: %s; Git Repo: %s", a.Title, a.Body, a.Head, a.Base, strings.Join(a.Labels, ", "), a.GitRepository.URL)
}

type GitWebHookArguments struct {
	ID          int64
	Owner       string
	Repo        *GitRepository
	URL         string
	ExistingURL string
	Secret      string
}

type GitFileContent struct {
	Type        string
	Encoding    string
	Size        int
	Name        string
	Path        string
	Content     string
	Sha         string
	Url         string
	GitUrl      string
	HtmlUrl     string
	DownloadUrl string
}

// PullRequestInfo describes a pull request that has been created
type PullRequestInfo struct {
	GitProvider          GitProvider
	PullRequest          *GitPullRequest
	PullRequestArguments *GitPullRequestArguments
}

// IsClosed returns true if the PullRequest has been closed
func (pr *GitPullRequest) IsClosed() bool {
	return pr.ClosedAt != nil
}

// NumberString returns the string representation of the Pull Request number or blank if its missing
func (pr *GitPullRequest) NumberString() string {
	n := pr.Number
	if n == nil {
		return ""
	}
	return "#" + strconv.Itoa(*n)
}

func CreateProvider(server auth.Server, git Gitter) (GitProvider, error) {
	if server.Kind == "" {
		server.Kind = SaasGitKind(server.URL)
	}
	if server.Kind == KindBitBucketCloud {
		return NewBitbucketCloudProvider(server, git)
	} else if server.Kind == KindBitBucketServer {
		return NewBitbucketServerProvider(server, git)
	} else if server.Kind == KindGitea {
		return NewGiteaProvider(server, git)
	} else if server.Kind == KindGitlab {
		return NewGitlabProvider(server, git)
	} else if server.Kind == KindGitFake {
		return NewFakeProvider(server)
	} else {
		return NewGitHubProvider(server, git)
	}
}

// GetHost returns the Git Provider hostname, e.g github.com
func GetHost(gitProvider GitProvider) (string, error) {
	if gitProvider == nil {
		return "", fmt.Errorf("no Git provider")
	}
	server := gitProvider.Server()
	if server.URL == "" {
		return "", fmt.Errorf("no Git provider server URL found")
	}
	url, err := url.Parse(server.URL)
	if err != nil {
		return "", fmt.Errorf("error parsing ")
	}
	return url.Host, nil
}

func ProviderAccessTokenURL(kind string, url string, username string) string {
	switch kind {
	case KindBitBucketCloud:
		return BitBucketCloudAccessTokenURL(url, username)
	case KindBitBucketServer:
		return BitBucketServerAccessTokenURL(url)
	case KindGitea:
		return GiteaAccessTokenURL(url)
	case KindGitlab:
		return GitlabAccessTokenURL(url)
	default:
		return GitHubAccessTokenURL(url)
	}
}

// GetOrganizations gets the organisation
func GetOrganizations(orgLister OrganisationLister, userName string) []string {
	var orgNames []string
	// Always include the username as a pseudo organization
	if userName != "" {
		orgNames = append(orgNames, userName)
	}

	orgs, _ := orgLister.ListOrganisations()
	for _, o := range orgs {
		if name := o.Login; name != "" {
			orgNames = append(orgNames, name)
		}
	}
	sort.Strings(orgNames)
	return orgNames
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

// ProviderURL returns the git provider URL
func (i *GitRepository) ProviderURL() string {
	scheme := i.Scheme
	if !strings.HasPrefix(scheme, "http") {
		scheme = "https"
	}
	return scheme + "://" + i.Host
}

// ToGitLabels converts the list of label names into an array of GitLabels
func ToGitLabels(names []string) []GitLabel {
	answer := []GitLabel{}
	for _, n := range names {
		answer = append(answer, GitLabel{Name: n})
	}
	return answer
}
