package decorators

import (
	"fmt"
	"net/http"
)

type jiraIssueConfig struct {
	CREDENTIALS struct {
		USERNAME string
		PASSWORD string
	}
	ENDPOINT struct {
		URL string
	}
	KEYS map[string]struct {
		DESTKEY string
		FIELD   string
	}
}

// jiraIssue fetch data using jira issue api
type jiraIssue struct {
	client http.Client
	config jiraIssueConfig
}

func (j jiraIssue) Decorate(commitMap *map[string]interface{}) (*map[string]interface{}, error) {
	var ID string

	switch v := (*commitMap)["jiraIssueId"].(type) {
	case string:
		ID = v
	case int64:
		ID = fmt.Sprintf("%d", v)
	default:
		return commitMap, nil
	}

	if ID == "" {
		return commitMap, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/2/issue/%s", j.config.ENDPOINT.URL, ID), nil)

	if err != nil {
		return commitMap, err
	}

	req.SetBasicAuth(j.config.CREDENTIALS.USERNAME, j.config.CREDENTIALS.PASSWORD)
	req.Header.Set("Content-Type", "application/json")

	return jSONResponse{&j.client, req, j.config.KEYS}.Decorate(commitMap)
}

func newJiraIssue(config jiraIssueConfig) Decorater {
	return jiraIssue{http.Client{}, config}
}
