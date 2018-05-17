package gits

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/suite"
	bitbucket "github.com/wbrefvem/go-bitbucket"
)

const (
	username = "test-user"
	orgName  = "test-org"
)

type BitbucketCloudProviderTestSuite struct {
	suite.Suite
	mux      *http.ServeMux
	server   *httptest.Server
	provider *BitbucketCloudProvider
}

var bitbucketRouter = util.Router{
	"/repositories/test-user": util.MethodMap{
		"GET": "repos.json",
	},
	"/repositories/test-user/test-repo": util.MethodMap{
		"GET":    "repos.test-repo.json",
		"DELETE": "repos.test-repo.nil.json",
		"PUT":    "repos.test-repo-renamed.json",
	},
	"/repositories/test-user/test-repo/forks": util.MethodMap{
		"POST": "repos.test-fork.json",
	},
	"/repositories/test-user/test-repo/pullrequests": util.MethodMap{
		"POST": "pullrequests.test-repo.json",
	},
	"/repositories/test-user/test-repo/pullrequests/3/": util.MethodMap{
		"GET": "pullrequests.test-repo-closed.json",
	},
	"/repositories/test-user/test-repo/pullrequests/1/commits": util.MethodMap{
		"GET": "pullrequests.test-user.test-repo.1.json",
	},
	"/repositories/test-user/test-repo/pullrequests/3/commits": util.MethodMap{
		"GET": "pullrequests.test-repo.commits.json",
	},
	"/repositories/test-user/test-repo/commit/5c8afc5/statuses": util.MethodMap{
		"GET": "repos.test-repo.statuses.json",
	},
	"/repositories/test-user/test-repo/commit/7793466f879b83f1bdd8f3fc3f761bc3cb61bc41": util.MethodMap{
		"GET": "repos.test-user.test-repo.commits.7793466f879b83f1bdd8f3fc3f761bc3cb61bc41.json",
	},
	"/repositories/test-user/test-repo/commit/bbc7b863a56144647a806646b73e3b43749decad": util.MethodMap{
		"GET": "repos.test-user.test-repo.commits.bbc7b863a56144647a806646b73e3b43749decad.json",
	},
	"/repositories/test-user/test-repo/pullrequests/1/merge": util.MethodMap{
		"POST": "pullrequests.test-repo.merged.json",
	},
	"/repositories/test-user/test-repo/hooks": util.MethodMap{
		"POST": "webhooks.example.json",
	},
	"/repositories/test-user/test-repo/issues": util.MethodMap{
		"POST": "issues.test-repo.issue-1.json",
		"GET":  "issues.test-repo.json",
	},
	"/repositories/test-user/test-repo/issues/1": util.MethodMap{
		"GET": "issues.test-repo.issue-1.json",
	},
	"/users/test-user": util.MethodMap{
		"GET": "users.test-user.json",
	},
}

func (suite *BitbucketCloudProviderTestSuite) SetupSuite() {
	suite.mux = http.NewServeMux()

	for path, methodMap := range bitbucketRouter {
		suite.mux.HandleFunc(path, util.GetMockAPIResponseFromFile("test_data/bitbucket", methodMap))
	}

	as := auth.AuthServer{
		URL:         "https://auth.example.com",
		Name:        "Test Auth Server",
		Kind:        "Oauth2",
		CurrentUser: "test-user",
	}
	ua := auth.UserAuth{
		Username: "test-user",
		ApiToken: "0123456789abdef",
	}

	bp, err := NewBitbucketCloudProvider(&as, &ua)

	suite.Require().NotNil(bp)
	suite.Require().Nil(err)

	var ok bool
	suite.provider, ok = bp.(*BitbucketCloudProvider)
	suite.Require().True(ok)
	suite.Require().NotNil(suite.provider)

	suite.server = httptest.NewServer(suite.mux)
	suite.Require().NotNil(suite.server)

	cfg := bitbucket.NewConfiguration()
	cfg.BasePath = suite.server.URL

	suite.provider.Client = bitbucket.NewAPIClient(cfg)
}

func (suite *BitbucketCloudProviderTestSuite) TestListRepositories() {

	repos, err := suite.provider.ListRepositories("test-user")

	suite.Require().Nil(err)
	suite.Require().NotNil(repos)

	suite.Require().Equal(len(repos), 2)

	for _, repo := range repos {
		suite.Require().NotNil(repo)
	}
}

func (suite *BitbucketCloudProviderTestSuite) TestGetRepository() {

	repo, err := suite.provider.GetRepository(
		suite.provider.Username,
		"test-repo",
	)

	suite.Require().NotNil(repo)
	suite.Require().Nil(err)

	suite.Require().Equal(repo.Name, "test-repo")
}

func (suite *BitbucketCloudProviderTestSuite) TestDeleteRepository() {

	err := suite.provider.DeleteRepository(
		suite.provider.Username,
		"test-repo",
	)

	suite.Require().Nil(err)
}

func (suite *BitbucketCloudProviderTestSuite) TestForkRepository() {

	fork, err := suite.provider.ForkRepository(
		suite.provider.Username,
		"test-repo",
		"",
	)

	suite.Require().NotNil(fork)
	suite.Require().Nil(err)

	suite.Require().Equal(fork.Name, "test-fork")
}

func (suite *BitbucketCloudProviderTestSuite) TestValidateRepositoryName() {

	err := suite.provider.ValidateRepositoryName(suite.provider.Username, "test-repo")

	suite.Require().NotNil(err)

	err = suite.provider.ValidateRepositoryName(suite.provider.Username, "foo-repo")

	suite.Require().Nil(err)
}

func (suite *BitbucketCloudProviderTestSuite) TestRenameRepository() {

	repo, err := suite.provider.RenameRepository(suite.provider.Username, "test-repo", "test-repo-renamed")

	suite.Require().Nil(err)
	suite.Require().NotNil(repo)

	suite.Require().Equal(repo.Name, "test-repo-renamed")
}

func (suite *BitbucketCloudProviderTestSuite) TestCreatePullRequest() {
	args := GitPullRequestArguments{
		Repo:  "test-repo",
		Head:  "83777f6",
		Base:  "77d0a923f297",
		Title: "Test Pull Request",
	}

	pr, err := suite.provider.CreatePullRequest(&args)

	suite.Require().NotNil(pr)
	suite.Require().Nil(err)
	suite.Require().Equal(*pr.State, "OPEN")
}

func (suite *BitbucketCloudProviderTestSuite) TestUpdatePullRequestStatus() {
	number := 3
	state := "OPEN"

	pr := &GitPullRequest{
		Repo:   "test-repo",
		Number: &number,
		State:  &state,
	}

	err := suite.provider.UpdatePullRequestStatus(pr)

	suite.Require().Nil(err)
}

func (suite *BitbucketCloudProviderTestSuite) TestGetPullRequest() {

	pr, err := suite.provider.GetPullRequest(
		"test-user",
		"test-repo",
		3,
	)

	suite.Require().Nil(err)
	suite.Require().Equal(*pr.Number, 3)
}

func (suite *BitbucketCloudProviderTestSuite) TestPullRequestCommits() {
	commits, err := suite.provider.GetPullRequestCommits("test-user", "test-repo", 1)

	suite.Require().Nil(err)
	suite.Require().Equal(len(commits), 2)
	suite.Require().Equal(commits[0].Author.Email, "test-user@gmail.com")
}

func (suite *BitbucketCloudProviderTestSuite) TestPullRequestLastCommitStatus() {

	pr := &GitPullRequest{
		Repo:          "test-repo",
		LastCommitSha: "5c8afc5",
	}
	lastCommitStatus, err := suite.provider.PullRequestLastCommitStatus(pr)

	suite.Require().Nil(err)
	suite.Require().NotEmpty(lastCommitStatus)
	suite.Require().Equal(lastCommitStatus, "in-progress")
}

func (suite *BitbucketCloudProviderTestSuite) TestListCommitStatus() {

	statuses, err := suite.provider.ListCommitStatus("test-user", "test-repo", "5c8afc5")

	suite.Require().Nil(err)
	suite.Require().NotNil(statuses)
	suite.Require().Equal(len(statuses), 2)

	for _, status := range statuses {
		suite.Require().NotEmpty(status.State)
		suite.Require().NotEmpty(status.URL)
	}
}

func (suite *BitbucketCloudProviderTestSuite) TestMergePullRequest() {

	id := 1
	pr := &GitPullRequest{
		Repo:   "test-repo",
		Number: &id,
	}
	err := suite.provider.MergePullRequest(pr, "Merging from unit tests")

	suite.Require().Nil(err)
}

func (suite *BitbucketCloudProviderTestSuite) TestCreateWebHook() {

	data := &GitWebHookArguments{
		Repo: "test-repo",
		URL:  "https://my-jenkins.example.com/bitbucket-webhook/",
	}
	err := suite.provider.CreateWebHook(data)

	suite.Require().Nil(err)
}

func (suite *BitbucketCloudProviderTestSuite) TestSearchIssues() {
	issues, err := suite.provider.SearchIssues("test-user", "test-repo", "")

	suite.Require().Nil(err)
	suite.Require().NotNil(issues)

	for _, issue := range issues {
		suite.Require().NotNil(issue)
	}
}

func (suite *BitbucketCloudProviderTestSuite) TestGetIssue() {
	issue, err := suite.provider.GetIssue("test-user", "test-repo", 1)

	suite.Require().Nil(err)
	suite.Require().NotNil(issue)
	suite.Require().Equal(*issue.Number, 1)
}

func (suite *BitbucketCloudProviderTestSuite) TestCreateIssue() {

	issueToCreate := &GitIssue{
		Title: "This is a test issue",
	}

	issue, err := suite.provider.CreateIssue("test-user", "test-repo", issueToCreate)

	suite.Require().Nil(err)
	suite.Require().NotNil(issue)
}

func (suite *BitbucketCloudProviderTestSuite) TestAddPRComment() {
	err := suite.provider.AddPRComment(nil, "")
	suite.Require().Nil(err)
}

func (suite *BitbucketCloudProviderTestSuite) TestCreateIssueComment() {
	err := suite.provider.CreateIssueComment("", "", 0, "")
	suite.Require().Nil(err)
}

func (suite *BitbucketCloudProviderTestSuite) TestUpdateRelease() {
	err := suite.provider.UpdateRelease("", "", "", nil)
	suite.Require().Nil(err)
}

func (suite *BitbucketCloudProviderTestSuite) TestUserInfo() {
	user := suite.provider.UserInfo("test-user")
	suite.Require().NotNil(user)
	suite.Require().Equal("test-user", user.Login)
}

func TestBitbucketCloudProviderTestSuite(t *testing.T) {
	suite.Run(t, new(BitbucketCloudProviderTestSuite))
}

func (suite *BitbucketCloudProviderTestSuite) TearDownSuite() {
	suite.server.Close()
}
