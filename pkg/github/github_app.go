package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// InstallationToken represents an installation from the github app
type InstallationToken struct {
	InstallationToken string `json:installationToken`
	InstallationID    string `json:installation_id`
	ExpireAt          string `json:expires_at`
}

// GetInstallationToken retrieves a github app installation token
func GetInstallationToken(githubAppUrl string, installationId string) (InstallationToken, error) {
	requestURL := fmt.Sprintf("%s/installation_token", githubAppUrl)

	params := url.Values{}
	params.Set("installation-id", installationId)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	httpUtils := util.HttpUtils{
		Client:     util.GetClient(),
		URL:        requestURL,
		Auth:       "",
		ReqBody:    []byte{},
		Headers:    &headers,
		HTTPMethod: http.MethodGet,
		ReqParams:  &params,
	}

	respBody, err := httpUtils.CallWithExponentialBackOff()
	if err != nil {
		return InstallationToken{}, errors.Wrapf(err, "error getting installation id via %s", requestURL)
	}

	var installationToken InstallationToken
	err = json.Unmarshal(respBody, &installationToken)
	if err != nil {
		return InstallationToken{}, errors.Wrap(err, "unmarshalling json message")
	}
	log.Logger().Debugf("github app installation token is %s", installationToken)

	return installationToken, nil
}
