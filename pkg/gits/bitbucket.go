package gits

type BitbucketProvider struct {
	Client *APIClient
}

func (b *BitbucketProvider) ListOrganisations() ([]GitOrganisation, error) {
	return nil, nil
}

func (b *BitbucketProvider) ListRepositories(org string) ([]*GitRepository, error) {
	return nil, nil
}

func (b *BitbucketProvider) CreateRepository(org string, name string, private bool) (*GitRepository, error) {
	return nil, nil, nil
}

func (b *BitbucketProvider) GetRepository(org string, name string) (*GitRepository, error) {
	return nil, nil
}

func (b *BitbucketProvider) DeleteRepository(org string, name string) error {
	return nil
}

func (b *BitbucketProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error) {
	return nil, nil
}

func (b *BitbucketProvider) RenameRepository(org string, name string, newName string) (*GitRepository, error) {
	return nil, nil
}

func (b *BitbucketProvider) ValidateRepositoryName(org string, name string) error {
	return nil
}

func (b *BitbucketProvider) CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error) {
	return nil, nil
}

func (b *BitbucketProvider) UpdatePullRequestStatus(pr *GitPullRequest) error {
	return nil
}

func (b *BitbucketProvider) PullRequestLastCommitStatus(pr *GitPullRequest) (string, error) {
	return nil, nil
}

func (b *BitbucketProvider) ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error) {
	return nil, nil
}

func (b *BitbucketProvider) MergePullRequest(pr *GitPullRequest, message string) error {
	return nil
}

func (b *BitbucketProvider) CreateWebHook(data *GitWebHookArguments) error {
	return nil
}

func (b *BitbucketProvider) GetIssue(org string, name string, number int) (*GitIssue, error) {
	return nil, nil
}

func (b *BitbucketProvider) CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error) {
	return nil, nil
}

func (b *BitbucketProvider) AddPRComment(pr *GitPullRequest, comment string) error {
	return nil
}

func (b *BitbucketProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	return nil
}

func (b *BibucketProvider) HasIssues() {
	return true
}

func (b *BitbucketProvider) IsGitHub() bool {
	return false
}

func (b *BitbuckProvider) IsGitea() bool {
	return false
}

func (b *BitbucketProvider) Kind() string {
	return "bitbucket"
}
