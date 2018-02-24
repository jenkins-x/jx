package decorators

import (
	"fmt"
	"net/http"

	"github.com/antham/chyle/chyle/apih"
)

type githubIssueConfig struct {
	CREDENTIALS struct {
		OAUTHTOKEN string
		OWNER      string
	}
	REPOSITORY struct {
		NAME string
	}
	KEYS map[string]struct {
		DESTKEY string
		FIELD   string
	}
}

// githubIssue fetch data using github issue api
type githubIssue struct {
	client http.Client
	config githubIssueConfig
}

func (g githubIssue) Decorate(commitMap *map[string]interface{}) (*map[string]interface{}, error) {
	var ID int64
	var ok bool

	if ID, ok = (*commitMap)["githubIssueId"].(int64); !ok {
		return commitMap, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", g.config.CREDENTIALS.OWNER, g.config.REPOSITORY.NAME, ID), nil)

	if err != nil {
		return commitMap, err
	}

	apih.SetHeaders(req, map[string]string{
		"Authorization": "token " + g.config.CREDENTIALS.OAUTHTOKEN,
		"Content-Type":  "application/json",
		"Accept":        "application/vnd.github.v3+json",
	})

	return jSONResponse{&g.client, req, g.config.KEYS}.Decorate(commitMap)
}

func newGithubIssue(config githubIssueConfig) Decorater {
	return githubIssue{http.Client{}, config}
}
