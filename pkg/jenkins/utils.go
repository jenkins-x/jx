package jenkins

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/util"
)

func GetJenkinsClient(url string, batch bool, configService *JenkinsConfigService) (*gojenkins.Jenkins, error) {
	if url == "" {
		return nil, errors.New("no JENKINS_URL env var set. Try running this command first:\n\n  eval $(gofabric8 bdd-env)\n")
	}
	username := os.Getenv("JENKINS_USERNAME")
	token := os.Getenv("JENKINS_TOKEN")
	bearerToken := os.Getenv("JENKINS_BEARER_TOKEN")

	tokenUrl := util.UrlJoin(url, "/me/configure")

	var err error
	config := JenkinsConfig{}
	auth := CreateJenkinsAuthFromEnvironment()
	if auth.IsInvalid() {
		// lets try load the current auth
		config, err = configService.LoadConfig()
		if err != nil {
			return nil, err
		}
		auths := config.FindAuths(url)
		if len(auths) > 1 {
			// TODO choose an auth
		}
		auth := config.FindAuth(url, username)
		if auth != nil {
			username = auth.Username
			token = auth.ApiToken
			bearerToken = auth.BearerToken

			if bearerToken == "" && (token == "" || username == "") {
				EditJenkinsAuth(url, configService, &config, auth)
			}
		} else {
			// lets create a new Auth
			auth = &JenkinsAuth{}
			EditJenkinsAuth(url, configService, &config, auth)
		}
	}
	if auth.IsInvalid() {
		fmt.Println("No $JENKINS_USERNAME and $JENKINS_TOKEN environment variables defined!\n")
		fmt.Printf("Please go to %s and click 'Show API Token' to get your API Token\n", tokenUrl)
		if batch {
			fmt.Println("Then run this command on your terminal and try again:\n")
			fmt.Println("export JENKINS_TOKEN=myApiToken\n")
			return nil, errors.New("No environment variables (JENKINS_USERNAME and JENKINS_TOKEN) or JENKINS_BEARER_TOKEN defined")
		}
	}

	jauth := &gojenkins.Auth{
		Username:    username,
		ApiToken:    token,
		BearerToken: bearerToken,
	}
	jenkins := gojenkins.NewJenkins(jauth, url)

	// handle insecure TLS for minishift
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	jenkins.SetHTTPClient(httpClient)
	return jenkins, nil
}

func EditJenkinsAuth(url string, configService *JenkinsConfigService, config *JenkinsConfig, auth *JenkinsAuth) error {
	// TODO let folks edit it

	config.SetAuth(url, *auth)
	return configService.SaveConfig(config)
}
