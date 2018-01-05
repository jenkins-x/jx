package jenkins

import (
	"crypto/tls"
	"errors"
	"net/http"
	"os"

	"github.com/jenkins-x/golang-jenkins"
	"fmt"
)

func GetJenkinsClient() (*gojenkins.Jenkins, error) {
	url := os.Getenv("JENKINS_URL")
	if url == "" {
		return nil, errors.New("no JENKINS_URL env var set. Try running this command first:\n\n  eval $(gofabric8 bdd-env)\n")
	}
	username := os.Getenv("JENKINS_USERNAME")
	token := os.Getenv("JENKINS_TOKEN")
	if token == "" {
		token = os.Getenv("JENKINS_PASSWORD")
	}
	bearerToken := os.Getenv("JENKINS_BEARER_TOKEN")

	fmt.Printf("url %s, user %s, token %s\n", url, username, token)
	if bearerToken == "" && (token == "" || username == "") {
		return nil, errors.New("no env vars JENKINS_BEARER_TOKEN or JENKINS_USERNAME and (JENKINS_TOKEN or JENKINS_PASSWORD ) set")
	}

	auth := &gojenkins.Auth{
		Username:    username,
		ApiToken:    token,
		BearerToken: bearerToken,
	}
	jenkins := gojenkins.NewJenkins(auth, url)

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
