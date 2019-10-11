package github

import (
	"encoding/json"
	"fmt"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"net/url"
)

type InstallationToken struct {
	InstallationToken string `json:installationToken`
	InstallationId    string `json:installation_id`
	ExpireAt          string `json:expires_at`
}

func GetInstallationToken(githubAppUrl string, installationId string) (InstallationToken, error) {
	requestUrl := fmt.Sprintf("%s/installation_token", githubAppUrl)

	params := url.Values{}
	params.Set("installation-id", installationId)

	respBody, err := util.CallWithExponentialBackOff(requestUrl, "", "GET", []byte{}, params)
	if err != nil {
		return InstallationToken{}, errors.Wrapf(err, "error getting installation id via %s", requestUrl)
	}

	var installationToken InstallationToken
	err = json.Unmarshal(respBody, &installationToken)
	if err != nil {
		return InstallationToken{}, errors.Wrap(err, "unmarshalling json message")
	}
	log.Logger().Debugf("github app installation token is %s", installationToken)

	return installationToken, nil
}
