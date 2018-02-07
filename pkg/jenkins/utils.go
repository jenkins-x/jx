package jenkins

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"github.com/jenkins-x/golang-jenkins"
	jenkauth "github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"io"
	"os"
)

func GetJenkinsClient(url string, batch bool, configService *jenkauth.AuthConfigService) (*gojenkins.Jenkins, error) {
	if url == "" {
		return nil, errors.New("no JENKINS_URL environment variable is set nor could a Jenkins service be found in the current namespace!\n")
	}
	tokenUrl := JenkinsTokenURL(url)

	auth := jenkauth.CreateAuthUserFromEnvironment("JENKINS")
	username := auth.Username
	var err error
	config := configService.Config()

	showForm := false
	if auth.IsInvalid() {
		// lets try load the current auth
		config, err = configService.LoadConfig()
		if err != nil {
			return nil, err
		}
		auths := config.FindUserAuths(url)
		if len(auths) > 1 {
			// TODO choose an auth
		}
		showForm = true
		a := config.FindUserAuth(url, username)
		if a != nil {
			if a.IsInvalid() {
				auth, err = EditUserAuth(url, configService, config, a, tokenUrl)
				if err != nil {
					return nil, err
				}
			} else {
				auth = *a
			}
		} else {
			// lets create a new Auth
			auth, err = EditUserAuth(url, configService, config, &auth, tokenUrl)
			if err != nil {
				return nil, err
			}
		}
	}

	if auth.IsInvalid() {
		if showForm {
			return nil, fmt.Errorf("No valid Username and API Token specified for Jenkins server: %s\n", url)
		} else {
			fmt.Println("No $JENKINS_USERNAME and $JENKINS_TOKEN environment variables defined!\n")
			PrintGetTokenFromURL(os.Stdout, tokenUrl)
			if batch {
				fmt.Println("Then run this command on your terminal and try again:\n")
				fmt.Println("export JENKINS_TOKEN=myApiToken\n")
				return nil, errors.New("No environment variables (JENKINS_USERNAME and JENKINS_TOKEN) or JENKINS_BEARER_TOKEN defined")
			}
		}
	}

	jauth := &gojenkins.Auth{
		Username:    auth.Username,
		ApiToken:    auth.ApiToken,
		BearerToken: auth.BearerToken,
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

func PrintGetTokenFromURL(out io.Writer, tokenUrl string) (int, error) {
	return fmt.Fprintf(out, "Please go to %s and click %s to get your API Token\n", util.ColorInfo(tokenUrl), util.ColorInfo("Show API Token"))
}

func JenkinsTokenURL(url string) string {
	tokenUrl := util.UrlJoin(url, "/me/configure")
	return tokenUrl
}

func EditUserAuth(url string, configService *jenkauth.AuthConfigService, config *jenkauth.AuthConfig, auth *jenkauth.UserAuth, tokenUrl string) (jenkauth.UserAuth, error) {
	fmt.Printf("\nTo be able to connect to the Jenkins server we need a username and API Token\n\n")
	fmt.Printf("Please go to %s and click %s to get your API Token\n", util.ColorInfo(tokenUrl), util.ColorInfo("Show API Token"))
	fmt.Printf("Then COPY the API token so that you can paste it into the form below:\n\n")

	defaultUsername := "admin"

	err := config.EditUserAuth("Jenkins", auth, defaultUsername, true)
	if err != nil {
		return *auth, err
	}
	err = configService.SaveUserAuth(url, auth)
	return *auth, err
}
