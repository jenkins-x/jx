package cve

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/util"
)

type result interface {
}

type VulnerabilityList struct {
	ImageDigest     string
	Vulnerabilities []Vulnerability
}

type Vulnerability struct {
	Fix      string
	Package  string
	Severity string
	URL      string
	Vuln     string
}

// AnchoreProvider implements CVEProvider interface for anchore.io
type AnchoreProvider struct {
	Client    *http.Client
	BasicAuth string
	BaseURL   string
}

func NewAnchoreProvider(server *auth.AuthServer, user *auth.UserAuth) (CVEProvider, error) {

	basicAuth := util.BasicAuth(user.Username, user.ApiToken)

	provider := AnchoreProvider{
		BaseURL:   server.URL,
		BasicAuth: basicAuth,
		Client:    http.DefaultClient,
	}

	return &provider, nil
}

func (a AnchoreProvider) GetImageVulnerabilityTable(table *table.Table, imageID string) error {
	subPath := fmt.Sprintf("/images/by_id/%s/vuln/%s", imageID, "os")

	var vList VulnerabilityList

	err := a.anchoreGet(subPath, &vList)
	if err != nil {
		return fmt.Errorf("error getting vulnerabilities for image %s: %v", imageID, err)
	}

	table.AddRow("Name", "Version", "Severity", "Vulnerability", "URL", "Package", "Fix")

	for _, v := range vList.Vulnerabilities {
		var sev string
		switch v.Severity {
		case "High":
			sev = util.ColorError(v.Severity)
		case "Medium":
			sev = util.ColorWarning(v.Severity)
		case "Low":
			sev = util.ColorStatus(v.Severity)
		}
		table.AddRow("foo", "0.0.1", sev, v.Vuln, v.URL, v.Package, v.Fix)
	}

	return nil
}

func (a AnchoreProvider) anchoreGet(subPath string, rs result) error {

	url := fmt.Sprintf("%s%s", a.BaseURL, subPath)
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Basic "+a.BasicAuth)

	resp, err := a.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting vulnerabilities from anchore engine %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("error response getting vulnerabilities from anchore engine: %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(data, &rs)
	if err != nil {
		return fmt.Errorf("error unmarshalling %v", err)
	}
	return nil
}
