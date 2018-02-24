package decorators

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v0"
)

func TestGithubIssue(t *testing.T) {
	config := githubIssueConfig{}
	config.CREDENTIALS.OAUTHTOKEN = "d41d8cd98f00b204e9800998ecf8427e"
	config.CREDENTIALS.OWNER = "user"
	config.REPOSITORY.NAME = "repository"
	config.KEYS = map[string]struct {
		DESTKEY string
		FIELD   string
	}{
		"MILESTONE": {
			"milestoneCreator",
			"milestone.creator.id",
		},
		"WHATEVER": {
			"whatever",
			"whatever",
		},
	}

	defer gock.Off()

	issueResponse, err := ioutil.ReadFile("fixtures/github-issue-fetch-response.json")

	assert.NoError(t, err, "Must read json fixture file")

	gock.New("https://api.github.com/repos/user/repository/issues/10000").
		MatchHeader("Authorization", "token d41d8cd98f00b204e9800998ecf8427e").
		MatchHeader("Content-Type", "application/json").
		HeaderPresent("Accept").
		Reply(200).
		JSON(string(issueResponse))

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	j := githubIssue{*client, config}

	result, err := j.Decorate(&map[string]interface{}{"test": "test", "githubIssueId": int64(10000)})

	expected := map[string]interface{}{
		"test":             "test",
		"githubIssueId":    int64(10000),
		"milestoneCreator": float64(1),
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, *result)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}

func TestGithubWithNoGithubIssueIdDefined(t *testing.T) {
	defer gock.Off()

	issueResponse, err := ioutil.ReadFile("fixtures/github-issue-fetch-response.json")

	assert.NoError(t, err, "Must read json fixture file")

	gock.New("https://api.github.com/repos/user/repository/issues/10000").
		Reply(200).
		JSON(string(issueResponse))

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	j := githubIssue{*client, githubIssueConfig{}}

	result, err := j.Decorate(&map[string]interface{}{"test": "test"})

	expected := map[string]interface{}{
		"test": "test",
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, *result)
	assert.False(t, gock.IsDone(), "Must have one pending request")
}

func TestGithubIssueWithAnErrorStatusCode(t *testing.T) {
	config := githubIssueConfig{}
	config.CREDENTIALS.OAUTHTOKEN = "d41d8cd98f00b204e9800998ecf8427e"
	config.CREDENTIALS.OWNER = "user"
	config.REPOSITORY.NAME = "repository"
	config.KEYS = map[string]struct {
		DESTKEY string
		FIELD   string
	}{
		"MILESTONE": {
			"milestoneCreator",
			"milestone.creator.id",
		},
		"WHATEVER": {
			"whatever",
			"whatever",
		},
	}

	defer gock.Off()

	gock.New("https://api.github.com/repos/user/repository/issues/10000").
		MatchHeader("Authorization", "token d41d8cd98f00b204e9800998ecf8427e").
		MatchHeader("Content-Type", "application/json").
		HeaderPresent("Accept").
		Reply(401).
		JSON(`{"message": "Bad credentials","documentation_url": "https://developer.github.com/v3"}`)

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	j := githubIssue{*client, config}

	_, err := j.Decorate(&map[string]interface{}{"test": "test", "githubIssueId": int64(10000)})

	assert.EqualError(t, err, `an error occurred when contacting remote api through https://api.github.com/repos/user/repository/issues/10000, status code 401, body {"message": "Bad credentials","documentation_url": "https://developer.github.com/v3"}`)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}

func TestGithubIssueWhenIssueIsNotFound(t *testing.T) {
	config := githubIssueConfig{}
	config.CREDENTIALS.OAUTHTOKEN = "d41d8cd98f00b204e9800998ecf8427e"
	config.CREDENTIALS.OWNER = "user"
	config.REPOSITORY.NAME = "repository"
	config.KEYS = map[string]struct {
		DESTKEY string
		FIELD   string
	}{
		"MILESTONE": {
			"milestoneCreator",
			"milestone.creator.id",
		},
		"WHATEVER": {
			"whatever",
			"whatever",
		},
	}

	defer gock.Off()

	gock.New("https://api.github.com/repos/user/repository/issues/10000").
		MatchHeader("Authorization", "token d41d8cd98f00b204e9800998ecf8427e").
		MatchHeader("Content-Type", "application/json").
		HeaderPresent("Accept").
		Reply(404).
		JSON(`{"message": "Not Found","documentation_url": "https://developer.github.com/v3"}`)

	client := &http.Client{Transport: &http.Transport{}}
	gock.InterceptClient(client)

	j := githubIssue{*client, config}

	result, err := j.Decorate(&map[string]interface{}{"test": "test", "githubIssueId": int64(10000)})

	expected := map[string]interface{}{
		"test":          "test",
		"githubIssueId": int64(10000),
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, *result)
	assert.True(t, gock.IsDone(), "Must have no pending requests")
}
