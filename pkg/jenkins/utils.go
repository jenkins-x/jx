package jenkins

import (
	"crypto/tls"
	"errors"
	"net/http"
	"os"

	"github.com/jenkins-x/golang-jenkins"
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
)

func GetJenkinsClient() (*gojenkins.Jenkins, error) {
	url := os.Getenv("JENKINS_URL")
	if url == "" {
		return nil, errors.New("no JENKINS_URL env var set. Try running this command first:\n\n  eval $(gofabric8 bdd-env)\n")
	}
	username := os.Getenv("JENKINS_USERNAME")
	token := os.Getenv("JENKINS_TOKEN")
	bearerToken := os.Getenv("JENKINS_BEARER_TOKEN")

	if bearerToken == "" && (token == "" || username == "") {
		tokenUrl := util.UrlJoin(url, "/me/configure")
		fmt.Println("No $JENKINS_USERNAME and $JENKINS_TOKEN environment variables defined!\n")
		fmt.Printf("Please go to %s and click 'Show API Token' to get your API Token\n", tokenUrl)
		fmt.Println("Then run this command on your terminal and try again:\n")
		fmt.Println("export JENKINS_TOKEN=myApiToken\n")
		return nil, errors.New("No environment variables (JENKINS_USERNAME and JENKINS_TOKEN) or JENKINS_BEARER_TOKEN defined")
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
