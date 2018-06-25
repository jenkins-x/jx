package gits

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"

	bitbucket "github.com/gfleury/go-bitbucket-v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

// BitbucketServerProvider implements GitProvider interface for a bitbucket server
type BitbucketServerProvider struct {
	Client   *bitbucket.APIClient
	Username string
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
	Git    Gitter
}

type projectsPage struct {
	Size          int                 `json:"size"`
	Limit         int                 `json:"limit"`
	Start         int                 `json:"start"`
	NextPageStart int                 `json:"nextPageStart"`
	IsLastPage    bool                `json:"isLastPage"`
	Values        []bitbucket.Project `json:"values"`
}

type commitsPage struct {
	Size          int                `json:"size"`
	Limit         int                `json:"limit"`
	Start         int                `json:"start"`
	NextPageStart int                `json:"nextPageStart"`
	IsLastPage    bool               `json:"isLastPage"`
	Values        []bitbucket.Commit `json:"values"`
}

type buildStatusesPage struct {
	Size       int                     `json:"size"`
	Limit      int                     `json:"limit"`
	Start      int                     `json:"start"`
	IsLastPage bool                    `json:"isLastPage"`
	Values     []bitbucket.BuildStatus `json:"values"`
}

type reposPage struct {
	Size          int                    `json:"size"`
	Limit         int                    `json:"limit"`
	Start         int                    `json:"start"`
	NextPageStart int                    `json:"nextPageStart"`
	IsLastPage    bool                   `json:"isLastPage"`
	Values        []bitbucket.Repository `json:"values"`
}

type pullrequestEndpointBranch struct {
	Name string `json:"name,omitempty"`
}

func NewBitbucketServerProvider(server *auth.AuthServer, user *auth.UserAuth, git Gitter) (GitProvider, error) {
	ctx := context.Background()
	apiKeyAuthContext := context.WithValue(ctx, bitbucket.ContextAccessToken, user.ApiToken)

	provider := BitbucketServerProvider{
		Server:   *server,
		User:     *user,
		Username: user.Username,
		Context:  apiKeyAuthContext,
		Git:      git,
	}

	cfg := bitbucket.NewConfiguration(server.URL + "/rest")
	provider.Client = bitbucket.NewAPIClient(apiKeyAuthContext, cfg)

	return &provider, nil
}

func BitbucketServerRepositoryToGitRepository(bRepo bitbucket.Repository) *GitRepository {
	var sshURL string
	var httpCloneURL string
	for _, link := range bRepo.Links.Clone {
		if link.Name == "ssh" {
			sshURL = link.Href
		}
	}
	isFork := false

	if httpCloneURL == "" {
		cloneLinks := bRepo.Links.Clone

		for _, link := range cloneLinks {
			if link.Name == "http" {
				httpCloneURL = link.Href
				if !strings.HasSuffix(httpCloneURL, ".git") {
					httpCloneURL += ".git"
				}
			}
		}
	}
	if httpCloneURL == "" {
		httpCloneURL = sshURL
	}

	return &GitRepository{
		Name:     bRepo.Name,
		HTMLURL:  bRepo.Links.Self[0].Href,
		CloneURL: httpCloneURL,
		SSHURL:   sshURL,
		Fork:     isFork,
	}
}

func (b *BitbucketServerProvider) GetRepository(org string, name string) (*GitRepository, error) {
	var repo bitbucket.Repository
	apiResponse, err := b.Client.DefaultApi.GetRepository(org, name)

	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(apiResponse.Values, &repo)
	if err != nil {
		return nil, err
	}

	return BitbucketServerRepositoryToGitRepository(repo), nil
}

func (b *BitbucketServerProvider) ListOrganisations() ([]GitOrganisation, error) {
	var orgsPage projectsPage
	orgsList := []GitOrganisation{}
	paginationOptions := make(map[string]interface{})

	paginationOptions["start"] = 0
	paginationOptions["limit"] = 25
	for {
		apiResponse, err := b.Client.DefaultApi.GetProjects(paginationOptions)
		if err != nil {
			return nil, err
		}

		err = mapstructure.Decode(apiResponse.Values, &orgsPage)
		if err != nil {
			return nil, err
		}

		for _, project := range orgsPage.Values {
			orgsList = append(orgsList, GitOrganisation{Login: project.Key})
		}

		if orgsPage.IsLastPage {
			break
		}
		paginationOptions["start"] = orgsPage.NextPageStart
	}

	return orgsList, nil
}

func (b *BitbucketServerProvider) ListRepositories(org string) ([]*GitRepository, error) {
	var reposPage reposPage
	repos := []*GitRepository{}
	paginationOptions := make(map[string]interface{})

	paginationOptions["start"] = 0
	paginationOptions["limit"] = 25

	for {
		apiResponse, err := b.Client.DefaultApi.GetRepositoriesWithOptions(org, paginationOptions)
		if err != nil {
			return nil, err
		}

		err = mapstructure.Decode(apiResponse.Values, &reposPage)
		if err != nil {
			return nil, err
		}

		for _, bRepo := range reposPage.Values {
			repos = append(repos, BitbucketServerRepositoryToGitRepository(bRepo))
		}

		if reposPage.IsLastPage {
			break
		}
		paginationOptions["start"] = reposPage.NextPageStart
	}

	return repos, nil
}

func (b *BitbucketServerProvider) CreateRepository(org, name string, private bool) (*GitRepository, error) {
	var repo bitbucket.Repository

	repoRequest := map[string]interface{}{
		"name":   name,
		"public": !private,
	}

	requestBody, err := json.Marshal(repoRequest)
	if err != nil {
		return nil, err
	}

	apiResponse, err := b.Client.DefaultApi.CreateRepositoryWithOptions(org, requestBody, []string{"application/json"})
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(apiResponse.Values, &repo)
	if err != nil {
		return nil, err
	}

	return BitbucketServerRepositoryToGitRepository(repo), nil
}

func (b *BitbucketServerProvider) DeleteRepository(org, name string) error {
	_, err := b.Client.DefaultApi.DeleteRepository(org, name)

	return err
}

func (b *BitbucketServerProvider) RenameRepository(org, name, newName string) (*GitRepository, error) {
	var repo bitbucket.Repository
	var options = map[string]interface{}{
		"name": newName,
	}

	requestBody, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}

	apiResponse, err := b.Client.DefaultApi.UpdateRepositoryWithOptions(org, name, requestBody, []string{"application/json"})
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(apiResponse.Values, &repo)
	if err != nil {
		return nil, err
	}

	return BitbucketServerRepositoryToGitRepository(repo), nil
}

func (b *BitbucketServerProvider) ValidateRepositoryName(org, name string) error {
	apiResponse, err := b.Client.DefaultApi.GetRepository(org, name)

	if apiResponse != nil && apiResponse.Response.StatusCode == 404 {
		return nil
	}

	if err == nil {
		return fmt.Errorf("repository %s/%s already exists", b.Username, name)
	}

	return err
}

func (b *BitbucketServerProvider) ForkRepository(originalOrg, name, destinationOrg string) (*GitRepository, error) {
	var repo bitbucket.Repository
	var apiResponse *bitbucket.APIResponse
	var options = map[string]interface{}{}

	if destinationOrg != "" {
		options["project"] = map[string]interface{}{
			"key": destinationOrg,
		}
	}

	requestBody, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}

	_, err = b.Client.DefaultApi.ForkRepository(originalOrg, name, requestBody, []string{"application/json"})
	if err != nil {
		return nil, err
	}

	// Wait up to 1 minute for the fork to be ready
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)

		if destinationOrg == "" {
			apiResponse, err = b.Client.DefaultApi.GetUserRepository(b.CurrentUsername(), name)
		} else {
			apiResponse, err = b.Client.DefaultApi.GetRepository(destinationOrg, name)
		}

		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(apiResponse.Values, &repo)
	if err != nil {
		return nil, err
	}

	return BitbucketServerRepositoryToGitRepository(repo), nil
}

func (b *BitbucketServerProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	var bPullRequest, bPR bitbucket.PullRequest
	var options = map[string]interface{}{
		"title":       data.Title,
		"description": data.Body,
		"state":       "OPEN",
		"open":        true,
		"closed":      false,
		"fromRef": map[string]interface{}{
			"id": data.Head,
			"repository": map[string]interface{}{
				"slug": data.GitRepositoryInfo.Name,
				"project": map[string]interface{}{
					"key": data.GitRepositoryInfo.Project,
				},
			},
		},
		"toRef": map[string]interface{}{
			"id": data.Base,
			"repository": map[string]interface{}{
				"slug": data.GitRepositoryInfo.Name,
				"project": map[string]interface{}{
					"key": data.GitRepositoryInfo.Project,
				},
			},
		},
	}

	requestBody, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}

	apiResponse, err := b.Client.DefaultApi.CreatePullRequestWithOptions(data.GitRepositoryInfo.Project, data.GitRepositoryInfo.Name, requestBody)
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(apiResponse.Values, &bPullRequest)
	if err != nil {
		return nil, err
	}

	// Wait up to 1 minute for the pull request to be ready
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)

		apiResponse, err = b.Client.DefaultApi.GetPullRequest(data.GitRepositoryInfo.Project, data.GitRepositoryInfo.Name, bPullRequest.ID)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(apiResponse.Values, &bPR)
	if err != nil {
		return nil, err
	}

	return &GitPullRequest{
		URL:    bPR.Links.Self[0].Href,
		Owner:  bPR.Author.User.Name,
		Repo:   bPR.ToRef.Repository.Name,
		Number: &bPR.ID,
		State:  &bPR.State,
	}, nil
}

func parseBitBucketServerURL(URL string) (string, string) {
	var projectKey, repoName, subString string
	var projectsIndex, reposIndex, repoEndIndex int

	if strings.HasSuffix(URL, ".git") {
		subString = strings.TrimSuffix(URL, ".git")
		reposIndex = strings.LastIndex(subString, "/")
		repoName = subString[reposIndex+1:]

		subString = strings.TrimSuffix(subString, "/"+repoName)
		projectsIndex = strings.LastIndex(subString, "/")
		projectKey = subString[projectsIndex+1:]

	} else {
		projectsIndex = strings.Index(URL, "projects/")
		subString = URL[projectsIndex+9:]
		projectKey = subString[:strings.Index(subString, "/")]

		reposIndex = strings.Index(subString, "repos/")
		subString = subString[reposIndex+6:]

		repoEndIndex = strings.Index(subString, "/")

		if repoEndIndex == -1 {
			repoName = subString
		} else {
			repoName = subString[:repoEndIndex]
		}
	}

	return projectKey, repoName
}

func getMergeCommitSHAFromPRActivity(prActivity map[string]interface{}) *string {
	var activity []map[string]interface{}
	var mergeCommit map[string]interface{}

	mapstructure.Decode(prActivity["values"], &activity)
	mapstructure.Decode(activity[0]["commit"], &mergeCommit)
	commitSHA := mergeCommit["id"].(string)

	return &commitSHA
}

func getLastCommitSHAFromPRCommits(prCommits map[string]interface{}) string {
	return getLastCommitFromPRCommits(prCommits).ID
}

func getLastCommitFromPRCommits(prCommits map[string]interface{}) *bitbucket.Commit {
	var commits []bitbucket.Commit
	mapstructure.Decode(prCommits["values"], &commits)
	return &commits[0]
}

func (b *BitbucketServerProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
	var bitbucketPR bitbucket.PullRequest
	var prCommits, prActivity map[string]interface{}

	prID := *pr.Number
	projectKey, repo := parseBitBucketServerURL(pr.URL)
	apiResponse, err := b.Client.DefaultApi.GetPullRequest(projectKey, repo, prID)
	if err != nil {
		return err
	}

	err = mapstructure.Decode(apiResponse.Values, &bitbucketPR)
	if err != nil {
		return err
	}

	pr.State = &bitbucketPR.State
	pr.Title = bitbucketPR.Title
	pr.Body = bitbucketPR.Description
	pr.Author = &GitUser{
		Login: bitbucketPR.Author.User.Name,
	}

	if bitbucketPR.State == "MERGED" {
		merged := true
		pr.Merged = &merged
		apiResponse, err := b.Client.DefaultApi.GetPullRequestActivity(projectKey, repo, prID)
		if err != nil {
			return err
		}

		mapstructure.Decode(apiResponse.Values, &prActivity)
		pr.MergeCommitSHA = getMergeCommitSHAFromPRActivity(prActivity)
	}
	diffURL := bitbucketPR.Links.Self[0].Href + "/diff"
	pr.DiffURL = &diffURL

	apiResponse, err = b.Client.DefaultApi.GetPullRequestCommits(projectKey, repo, prID)
	if err != nil {
		return err
	}
	mapstructure.Decode(apiResponse.Values, &prCommits)
	pr.LastCommitSha = getLastCommitSHAFromPRCommits(prCommits)

	return nil
}

func (b *BitbucketServerProvider) GetPullRequest(owner string, repo *GitRepositoryInfo, number int) (*GitPullRequest, error) {
	var bPR bitbucket.PullRequest

	apiResponse, err := b.Client.DefaultApi.GetPullRequest(repo.Project, repo.Name, number)
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(apiResponse.Values, &bPR)
	if err != nil {
		return nil, err
	}

	author := &GitUser{
		URL:   bPR.Author.User.Links.Self[0].Href,
		Login: bPR.Author.User.Slug,
		Name:  bPR.Author.User.Name,
		Email: bPR.Author.User.Email,
	}

	return &GitPullRequest{
		URL:    bPR.Links.Self[0].Href,
		Owner:  bPR.Author.User.Name,
		Repo:   bPR.ToRef.Repository.Name,
		Number: &bPR.ID,
		State:  &bPR.State,
		Author: author,
	}, nil
}

func convertBitBucketCommitToGitCommit(bCommit *bitbucket.Commit, repo *GitRepositoryInfo) *GitCommit {
	return &GitCommit{
		SHA:     bCommit.ID,
		Message: bCommit.Message,
		Author: &GitUser{
			Login: bCommit.Author.Name,
			Name:  bCommit.Author.DisplayName,
			Email: bCommit.Author.Email,
		},
		URL: repo.URL + "/commits/" + bCommit.ID,
		Committer: &GitUser{
			Login: bCommit.Committer.Name,
			Name:  bCommit.Committer.DisplayName,
			Email: bCommit.Committer.Email,
		},
	}
}

func (b *BitbucketServerProvider) GetPullRequestCommits(owner string, repository *GitRepositoryInfo, number int) ([]*GitCommit, error) {
	var commitsPage commitsPage
	commits := []*GitCommit{}
	paginationOptions := make(map[string]interface{})

	paginationOptions["start"] = 0
	paginationOptions["limit"] = 25
	for {
		apiResponse, err := b.Client.DefaultApi.GetPullRequestCommitsWithOptions(repository.Project, repository.Name, number, paginationOptions)
		if err != nil {
			return nil, err
		}

		err = mapstructure.Decode(apiResponse.Values, &commitsPage)
		if err != nil {
			return nil, err
		}

		for _, commit := range commitsPage.Values {
			commits = append(commits, convertBitBucketCommitToGitCommit(&commit, repository))
		}

		if commitsPage.IsLastPage {
			break
		}
		paginationOptions["start"] = commitsPage.NextPageStart
	}

	return commits, nil
}

func (b *BitbucketServerProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	var prCommits map[string]interface{}
	var buildStatusesPage buildStatusesPage

	projectKey, repo := parseBitBucketServerURL(pr.URL)
	apiResponse, err := b.Client.DefaultApi.GetPullRequestCommits(projectKey, repo, *pr.Number)
	if err != nil {
		return "", err
	}
	mapstructure.Decode(apiResponse.Values, &prCommits)
	lastCommit := getLastCommitFromPRCommits(prCommits)
	lastCommitSha := lastCommit.ID

	apiResponse, err = b.Client.DefaultApi.GetCommitBuildStatuses(lastCommitSha)
	if err != nil {
		return "", err
	}

	mapstructure.Decode(apiResponse.Values, &buildStatusesPage)
	if buildStatusesPage.Size == 0 {
		return "SUCCESSFUL", nil
	}

	for _, buildStatus := range buildStatusesPage.Values {
		if time.Unix(buildStatus.DateAdded, 0).After(time.Unix(lastCommit.CommitterTimestamp, 0)) {
			return buildStatus.State, nil
		}
	}

	return "SUCCESSFUL", nil
}

func (b *BitbucketServerProvider) ListCommitStatus(org, repo, sha string) ([]*GitRepoStatus, error) {
	var buildStatusesPage buildStatusesPage
	statuses := []*GitRepoStatus{}

	for {
		apiResponse, err := b.Client.DefaultApi.GetCommitBuildStatuses(sha)
		if err != nil {
			return nil, err
		}

		mapstructure.Decode(apiResponse.Values, &buildStatusesPage)

		for _, buildStatus := range buildStatusesPage.Values {
			statuses = append(statuses, convertBitBucketBuildStatusToGitStatus(&buildStatus))
		}

		if buildStatusesPage.IsLastPage {
			break
		}
	}

	return statuses, nil
}

func convertBitBucketBuildStatusToGitStatus(buildStatus *bitbucket.BuildStatus) *GitRepoStatus {
	return &GitRepoStatus{
		ID:          buildStatus.Key,
		URL:         buildStatus.Url,
		State:       buildStatus.State,
		TargetURL:   buildStatus.Url,
		Description: buildStatus.Description,
	}
}

func (b *BitbucketServerProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	var currentPR bitbucket.PullRequest
	projectKey, repo := parseBitBucketServerURL(pr.URL)
	queryParams := map[string]interface{}{}

	apiResponse, err := b.Client.DefaultApi.GetPullRequest(projectKey, repo, *pr.Number)
	if err != nil {
		return err
	}

	mapstructure.Decode(apiResponse.Values, &currentPR)
	queryParams["version"] = currentPR.Version

	var options = map[string]interface{}{
		"message": message,
	}

	requestBody, err := json.Marshal(options)
	if err != nil {
		return err
	}

	apiResponse, err = b.Client.DefaultApi.Merge(projectKey, repo, *pr.Number, queryParams, requestBody, []string{"application/json"})
	if err != nil {
		return err
	}

	return nil
}

func (b *BitbucketServerProvider) CreateWebHook(data *GitWebHookArguments) error {
	projectKey, repo := parseBitBucketServerURL(data.Repo.URL)

	var options = map[string]interface{}{
		"url":    data.URL,
		"name":   "Jenkins X Web Hook",
		"active": true,
		"events": []string{"repo:refs_changed", "repo:modified", "repo:forked", "repo:comment:added", "repo:comment:edited", "repo:comment:deleted", "pr:opened", "pr:reviewer:approved", "pr:reviewer:unapproved", "pr:reviewer:needs_work", "pr:merged", "pr:declined", "pr:deleted", "pr:comment:added", "pr:comment:edited", "pr:comment:deleted"},
	}

	if data.Secret != "" {
		options["configuration"] = map[string]interface{}{
			"secret": data.Secret,
		}
	}

	requestBody, err := json.Marshal(options)
	if err != nil {
		return err
	}

	_, err = b.Client.DefaultApi.CreateWebhook(projectKey, repo, requestBody, []string{"application/json"})

	return err
}

func (b *BitbucketServerProvider) SearchIssues(org string, name string, query string) ([]*GitIssue, error) {

	gitIssues := []*GitIssue{}

	log.Warn("Searching issues on bitbucket server is not supported at this moment")

	return gitIssues, nil
}

func (b *BitbucketServerProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error) {
	issues, err := b.SearchIssues(org, name, "")
	if err != nil {
		return issues, err
	}
	return FilterIssuesClosedSince(issues, t), nil
}

func (b *BitbucketServerProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {

	log.Warn("Finding an issue on bitbucket server is not supported at this moment")
	return &GitIssue{}, nil
}

func (b *BitbucketServerProvider) IssueURL(org string, name string, number int, isPull bool) string {
	serverPrefix := b.Server.URL
	if strings.Index(serverPrefix, "://") < 0 {
		serverPrefix = "https://" + serverPrefix
	}
	path := "issues"
	if isPull {
		path = "pull"
	}
	url := util.UrlJoin(serverPrefix, org, name, path, strconv.Itoa(number))
	return url
}

func (b *BitbucketServerProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {

	log.Warn("Creating an issue on bitbucket server is not suuported at this moment")
	return &GitIssue{}, nil
}

func (b *BitbucketServerProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	log.Warn("Bitbucket Server doesn't support adding PR comments via the REST API")
	return nil
}

func (b *BitbucketServerProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	log.Warn("Bitbucket Server doesn't support adding issue comments via the REST API")
	return nil
}

func (b *BitbucketServerProvider) HasIssues() bool {
	return true
}

func (b *BitbucketServerProvider) IsGitHub() bool {
	return false
}

func (b *BitbucketServerProvider) IsGitea() bool {
	return false
}

func (b *BitbucketServerProvider) IsBitbucketCloud() bool {
	return false
}

func (b *BitbucketServerProvider) IsBitbucketServer() bool {
	return true
}

func (b *BitbucketServerProvider) Kind() string {
	return "bitbucketserver"
}

// Exposed by Jenkins plugin; this one is for https://wiki.jenkins.io/display/JENKINS/BitBucket+Plugin
func (b *BitbucketServerProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/bitbucket-scmsource-hook/notify"
}

func (b *BitbucketServerProvider) Label() string {
	return b.Server.Label()
}

func (b *BitbucketServerProvider) ServerURL() string {
	return b.Server.URL
}

func (b *BitbucketServerProvider) CurrentUsername() string {
	return b.Username
}

func (b *BitbucketServerProvider) UserAuth() auth.UserAuth {
	return b.User
}

func (b *BitbucketServerProvider) UserInfo(username string) *GitUser {
	var user bitbucket.UserWithLinks
	apiResponse, err := b.Client.DefaultApi.GetUser(username)
	if err != nil {
		log.Error("Unable to fetch user info for " + username + " due to " + err.Error() + "\n")
		return nil
	}
	err = mapstructure.Decode(apiResponse.Values, &user)

	return &GitUser{
		Login: username,
		Name:  user.DisplayName,
		Email: user.Email,
		URL:   user.Links.Self[0].Href,
	}
}

func (b *BitbucketServerProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error {
	log.Warn("Bitbucket Server doesn't support releases")
	return nil
}

func (p *BitbucketServerProvider) ListReleases(org string, name string) ([]*GitRelease, error) {
	answer := []*GitRelease{}
	log.Warn("Bitbucket Server doesn't support releases")
	return answer, nil
}

func BitBucketServerAccessTokenURL(url string) string {
	// TODO with github we can default the scopes/flags we need on a token via adding
	// ?scopes=repo,read:user,user:email,write:repo_hook
	//
	// is there a way to do that for bitbucket?
	return util.UrlJoin(url, "/plugins/servlet/access-tokens/manage")
}
