package pipline_events

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"fmt"

	"bytes"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
)

type result interface {
}

type Index struct {
	Created bool
	Id      string `json:"_id,omitempty"`
}

// ElasticsearchProvider implements PipelineEventsProvider interface for elasticsearch
type ElasticsearchProvider struct {
	Client    *http.Client
	BasicAuth string
	BaseURL   string
}

func NewElasticsearchProvider(server *auth.AuthServer, user *auth.UserAuth) (PipelineEventsProvider, error) {

	basicAuth := util.BasicAuth(user.Username, user.Password)

	provider := ElasticsearchProvider{
		BaseURL:   server.URL,
		BasicAuth: basicAuth,
		Client:    http.DefaultClient,
	}

	return &provider, nil
}

func (e ElasticsearchProvider) SendActivity(a *v1.PipelineActivity) error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	var index *Index

	err = e.post("activities", string(a.UID), data, &index)
	if err != nil {
		return err
	}

	if index.Id == "" {
		return fmt.Errorf("activity %s not created, no elasticsearch id returned from POST\n", a.Name)
	}
	return nil
}

func (e ElasticsearchProvider) post(index, indexID string, body []byte, rs result) error {

	url := fmt.Sprintf("%s/%s/event/%s", e.BaseURL, index, indexID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Add("Authorization", "Basic "+e.BasicAuth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error POSTing to elasticsearch %v", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("error response POSTing to elasticsearch: %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(data, &rs)
	if err != nil {
		return fmt.Errorf("error unmarshalling %v", err)
	}
	return nil
}
