package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	JenkinsCookieName    = "JSESSIONID"
	JenkinsVersionHeader = "X-Jenkins"
)

var JenkinsReferenceVersion = semver.Version{Major: 2, Minor: 140, Patch: 0}

var (
	create_jenkins_user_long = templates.LongDesc(`
		Creates a new user and API Token for the current Jenkins server
`)

	create_jenkins_user_example = templates.Examples(`
		# Add a new API Token for a user for the current Jenkins server
        # prompting the user to find and enter the API Token
		jx create jenkins token someUserName

		# Add a new API Token for a user for the current Jenkins server
 		# using browser automation to login to the Git server
		# with the username an password to find the API Token
		jx create jenkins token -p somePassword someUserName	
	`)
)

// CreateJenkinsUserOptions the command line options for the command
type CreateJenkinsUserOptions struct {
	CreateOptions

	ServerFlags ServerFlags
	Username    string
	Password    string
	ApiToken    string
	BearerToken string
	Timeout     string
	UseBrowser  bool
}

// NewCmdCreateJenkinsUser creates a command
func NewCmdCreateJenkinsUser(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateJenkinsUserOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "token [username]",
		Short:   "Adds a new username and API token for a Jenkins server",
		Aliases: []string{"api-token"},
		Long:    create_jenkins_user_long,
		Example: create_jenkins_user_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	options.ServerFlags.addGitServerFlags(cmd)
	cmd.Flags().StringVarP(&options.ApiToken, "api-token", "t", "", "The API Token for the user")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "The User password to try automatically create a new API Token")
	cmd.Flags().StringVarP(&options.Timeout, "timeout", "", "", "The timeout if using REST to generate the API token (by passing username and password)")
	cmd.Flags().BoolVarP(&options.UseBrowser, "browser", "", false, "Use REST calls to automatically find the API token if the user and password are known")

	return cmd
}

// Run implements the command
func (o *CreateJenkinsUserOptions) Run() error {
	args := o.Args
	if len(args) > 0 {
		o.Username = args[0]
	}
	if len(args) > 1 {
		o.ApiToken = args[1]
	}
	kubeClient, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return fmt.Errorf("error connecting to Kubernetes cluster: %v", err)
	}

	authConfigSvc, err := o.CreateJenkinsAuthConfigService(kubeClient, ns)
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	var server *auth.AuthServer
	if o.ServerFlags.IsEmpty() {
		url := ""
		url, err = o.findService(kube.ServiceJenkins)
		if err != nil {
			return err
		}
		server = config.GetOrCreateServer(url)
	} else {
		server, err = o.findServer(config, &o.ServerFlags, "jenkins server", "Try installing one via: jx create team", false)
		if err != nil {
			return err
		}
	}

	// TODO add the API thingy...
	if o.Username == "" {
		return fmt.Errorf("No Username specified")
	}

	userAuth := config.GetOrCreateUserAuth(server.URL, o.Username)
	if o.ApiToken != "" {
		userAuth.ApiToken = o.ApiToken
	}

	if o.BearerToken != "" {
		userAuth.BearerToken = o.BearerToken
	}

	if o.Password != "" {
		userAuth.Password = o.Password
	}

	if userAuth.IsInvalid() && o.Password != "" && o.UseBrowser {
		err := o.getAPITokenFromREST(server.URL, userAuth)
		if err != nil {
			return err
		}
	}

	if userAuth.IsInvalid() {
		f := func(username string) error {
			jenkins.PrintGetTokenFromURL(o.Out, jenkins.JenkinsTokenURL(server.URL))
			log.Infof("Then COPY the token and enter in into the form below:\n\n")
			return nil
		}

		err = config.EditUserAuth("Jenkins", userAuth, o.Username, false, o.BatchMode, f, o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		if userAuth.IsInvalid() {
			return fmt.Errorf("You did not properly define the user authentication!")
		}
	}

	config.CurrentServer = server.URL
	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}

	// now lets create a secret for it so we can perform incluster interactions with Jenkins
	s, err := kubeClient.CoreV1().Secrets(o.currentNamespace).Get(kube.SecretJenkins, metav1.GetOptions{})
	if err != nil {
		return err
	}
	s.Data[kube.JenkinsAdminApiToken] = []byte(userAuth.ApiToken)
	s.Data[kube.JenkinsBearTokenField] = []byte(userAuth.BearerToken)
	s.Data[kube.JenkinsAdminUserField] = []byte(userAuth.Username)
	_, err = kubeClient.CoreV1().Secrets(o.currentNamespace).Update(s)
	if err != nil {
		return err
	}

	log.Infof("Created user %s API Token for Jenkins server %s at %s\n",
		util.ColorInfo(o.Username), util.ColorInfo(server.Name), util.ColorInfo(server.URL))
	return nil
}

func (o *CreateJenkinsUserOptions) getAPITokenFromREST(serverURL string, userAuth *auth.UserAuth) error {
	var ctx context.Context
	var cancel context.CancelFunc
	if o.Timeout != "" {
		duration, err := time.ParseDuration(o.Timeout)
		if err != nil {
			return errors.Wrap(err, "parsing the timeout value")
		}
		ctx, cancel = context.WithTimeout(context.Background(), duration)
		defer cancel()
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	log.Infoln("Generating the API token...")
	decorator, err := loginLegacy(ctx, serverURL, userAuth.Username, o.Password)
	if err != nil {
		// TODO might be modern realm; try: req.SetBasicAuth(userAuth.Username, o.Password)
		return errors.Wrap(err, "logging in")
	}
	decorator = checkForCrumb(ctx, serverURL, decorator)
	token, err := generateNewAPIToken(ctx, serverURL, decorator)
	if err != nil {
		return errors.Wrap(err, "generating the API token")
	}
	if token == "" {
		return errors.New("received an empty API token")
	}
	userAuth.ApiToken = token
	return nil
}

func loginLegacy(ctx context.Context, serverURL string, username string, password string) (func(req *http.Request), error) {
	client := http.Client{
		// https://stackoverflow.com/a/38150816/12916 Jenkins returns a 303, but you cannot actually follow it
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/j_security_check?j_username=%s&j_password=%s", serverURL, url.QueryEscape(username), url.QueryEscape(password)), nil)
	if err != nil {
		return nil, errors.Wrap(err, "building request to log in")
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "execute log in")
	}
	defer resp.Body.Close()
	cookies := resp.Cookies()
	decorator := func(req *http.Request) {
		for _, c := range cookies {
			req.AddCookie(c)
		}
	}
	// We get the same response even if Jenkins is actually using a modern security realm, so verify it:
	err = verifyLogin(ctx, serverURL, decorator)
	if err != nil {
		return nil, errors.Wrap(err, "cookies did not work; bad login or not using legacy security realm")
	}
	log.Infof("Logged in %s to Jenkins server at %s via legacy security realm\n",
		util.ColorInfo(username), util.ColorInfo(serverURL))
	return decorator, nil
}

func verifyLogin(ctx context.Context, serverURL string, decorator func (req *http.Request)) error {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/me/api/json?tree=id", serverURL), nil)
	if err != nil {
		return errors.Wrap(err, "building request to verify login")
	}
	req = req.WithContext(ctx)
	decorator(req)
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "execute verify login")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}
	return nil
}

func checkForCrumb(ctx context.Context, serverURL string, decorator func (req *http.Request)) func (req *http.Request) {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)", serverURL), nil)
	if err != nil {
		log.Warnf("Failed to build request to check for crumb: %s\n", err)
		return decorator
	}
	req = req.WithContext(ctx)
	decorator(req)
	resp, err := client.Do(req)
	if err != nil {
		log.Warnf("Failed to execute request to check for crumb: %s\n", err)
		return decorator
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		log.Infof("Enable CSRF protection at: %s/configureSecurity/\n", serverURL)
		return decorator
	} else if resp.StatusCode != 200 {
		log.Warnf("Could not find CSRF crumb: %d %s\n", resp.StatusCode, resp.Status)
		return decorator
	}
	if resp.StatusCode != 200 {
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Failed to read crumb: %s\n", err)
		return decorator
	}
	crumbPieces := strings.SplitN(string(body), ":", 2)
	if len(crumbPieces) != 2 {
		log.Warnf("Malformed crumb: %s\n", body)
		return decorator
	}
	log.Infof("Obtained crumb\n")
	return func(req *http.Request) {
		decorator(req)
		req.Header.Add(crumbPieces[0], crumbPieces[1])
	}
	return decorator
}

func generateNewAPIToken(ctx context.Context, serverURL string, decorator func(req *http.Request)) (string, error) {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodPost, jenkins.JenkinsNewTokenURL(serverURL), nil)
	if err != nil {
		return "", errors.Wrap(err, "building request to generate the API token")
	}
	req = req.WithContext(ctx)
	decorator(req)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "execute generate API token request")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading API token from response body")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("generate API token status code: %d, error: %s", resp.StatusCode, string(body))
	}

	type TokenData struct {
		TokenName  string `json:"tokenName"`
		TokenUuid  string `json:"tokenUuid"`
		TokenValue string `json:"tokenValue"`
	}

	type TokenResponse struct {
		Status string    `json:"status"`
		Data   TokenData `json:"data"`
	}
	tokenResponse := &TokenResponse{}
	if err := json.Unmarshal(body, tokenResponse); err != nil {
		return "", errors.Wrap(err, "parsing the API token from response")
	}
	return tokenResponse.Data.TokenValue, nil
}
