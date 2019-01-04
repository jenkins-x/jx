package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/chromedp/chromedp/runner"
	"github.com/hpcloud/tail"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	jxlog "github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	defaultNamespace       = "jx"
	UserOnboardingEndpoint = "/api/v1/users"
	SsoCookieName          = "sso-core"
)

// Login holds the login information
type Login struct {
	Data UserLoginInfo `form:"data,omitempty" json:"data,omitempty" yaml:"data,omitempty" xml:"data,omitempty"`
}

// UserLoginInfo user login information
type UserLoginInfo struct {
	// The Kubernetes API server public CA data
	Ca string `form:"ca,omitempty" json:"ca,omitempty" yaml:"ca,omitempty" xml:"ca,omitempty"`
	// The login username of the user
	Login string `form:"login,omitempty" json:"login,omitempty" yaml:"login,omitempty" xml:"login,omitempty"`
	// The Kubernetes API server address
	Server string `form:"server,omitempty" json:"server,omitempty" yaml:"server,omitempty" xml:"server,omitempty"`
	// The login token of the user
	Token string `form:"token,omitempty" json:"token,omitempty" yaml:"token,omitempty" xml:"token,omitempty"`
}

// LoginOptions options for login command
type LoginOptions struct {
	CommonOptions

	URL  string
	Team string
}

var (
	login_long = templates.LongDesc(`
		Onboards an user into the CloudBees application and configures the Kubernetes client configuration.

		A CloudBess app can be created as an addon with 'jx create addon cloudbees'`)

	login_example = templates.Examples(`
		# Onboard into CloudBees application
		jx login -u https://cloudbees-app-url 
	
		# Onboard into CloudBees application and switched to team 'cheese'
		jx login -u https://cloudbees-app-url -t cheese
		`)
)

func NewCmdLogin(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &LoginOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,

			Out: out,
			Err: errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "login",
		Short:   "Onboard an user into the CloudBees application",
		Long:    login_long,
		Example: login_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.URL, "url", "u", "", "The URL of the CloudBees application")
	cmd.Flags().StringVarP(&options.Team, "team", "t", "", "The team to use upon login")

	return cmd
}

func (o *LoginOptions) Run() error {

	_, err := url.ParseRequestURI(o.URL)
	if err != nil {
		return errors.Wrap(err, "validation failed for URL, ensure URL is well formed including scheme, i.e. https://foo.com")
	}

	// ensure base set of binaries are installed which are required by jx
	err = o.installRequirements("")
	if err != nil {
		return errors.Wrap(err, "installing required binaries")
	}

	userLoginInfo, err := o.Login()
	if err != nil {
		return errors.Wrap(err, "logging into the CloudBees application")
	}

	err = o.Kube().UpdateConfig(defaultNamespace, userLoginInfo.Server, userLoginInfo.Ca, userLoginInfo.Login, userLoginInfo.Token)
	if err != nil {
		return errors.Wrap(err, "updating the ~/kube/config file")
	}

	jxlog.Infof("You are %s. You credentials are stored in %s file.\n",
		util.ColorInfo("successfully logged in"), util.ColorInfo("~/.kube/config"))

	teamOptions := TeamOptions{
		CommonOptions: o.CommonOptions,
	}
	teamOptions.Args = []string{o.Team}
	err = teamOptions.Run()
	if err != nil {
		return errors.Wrap(err, "switching team")
	}

	return nil
}

func (o *LoginOptions) Login() (*UserLoginInfo, error) {
	url := o.URL
	if url == "" {
		return nil, errors.New("please provide the URL of the CloudBees application in '--url' option")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	userDataDir, err := ioutil.TempDir("", "jx-login-chrome-userdata-dir")
	if err != nil {
		return nil, errors.Wrap(err, "creating the chrome user data dir")
	}
	defer os.RemoveAll(userDataDir)

	netLogFile := filepath.Join(userDataDir, "net-logs.json")
	options := func(m map[string]interface{}) error {
		m["start-url"] = o.URL
		m["user-data-dir"] = userDataDir
		m["log-net-log"] = netLogFile
		m["net-log-capture-mode"] = "IncludeCookiesAndCredentials"
		m["v"] = 1
		return nil
	}

	r, err := runner.New(options)
	if err != nil {
		return nil, errors.Wrap(err, "creating chrome runner")
	}

	err = r.Start(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "starting chrome")
	}

	t, err := tail.TailFile(netLogFile, tail.Config{
		Follow: true,
		Poll: true,  // ionotify does not work on all platforms and the cost for this is not significantly high
		Logger: log.New(ioutil.Discard, "", log.LstdFlags)})
	if err != nil {
		return nil, errors.Wrap(err, "reading the netlog file")
	}
	cookie := ""
	pattern := fmt.Sprintf("%s=", SsoCookieName)
	for line := range t.Lines {
		if strings.Contains(line.Text, pattern) {
			cookie = ExtractSsoCookie(line.Text)
			break
		}
	}

	if cookie == "" {
		return nil, errors.New("failed to log into the CloudBees application")
	}

	userLoginInfo, err := o.OnboardUser(cookie)
	if err != nil {
		return nil, errors.Wrap(err, "onboarding user")
	}

	err = r.Shutdown(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "shutting down Chrome")
	}

	err = r.Wait()
	if err != nil {
		return nil, errors.Wrap(err, "waiting for Chrome  to exit")
	}

	return userLoginInfo, nil
}

func (o *LoginOptions) OnboardUser(cookie string) (*UserLoginInfo, error) {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodPost, o.onboardingURL(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "building onboarding request")
	}
	req.Header.Add("Accept", "application/json")
	if cookie == "" {
		return nil, errors.New("empty SSO cookie")
	}
	ssoCookie := http.Cookie{Name: SsoCookieName, Value: cookie}
	req.AddCookie(&ssoCookie)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "executing onboarding request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading user onboarding information from response body")
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("user onboarding status code: %d, error: %s", resp.StatusCode, string(body))
	}

	login := &Login{}
	if err := json.Unmarshal(body, login); err != nil {
		return nil, errors.Wrap(err, "parsing the login information from response")
	}

	return &login.Data, nil
}

func (o *LoginOptions) onboardingURL() string {
	url := o.URL
	if strings.HasSuffix(url, "/") {
		url = strings.TrimSuffix(url, "/")
	}
	return url + UserOnboardingEndpoint
}

func ExtractSsoCookie(text string) string {
	cookiePattern := fmt.Sprintf("%s=", SsoCookieName)
	start := strings.Index(text, cookiePattern)
	if start < 0 {
		return ""
	}
	end := -1
	cookieStart := start + len(cookiePattern)
	for i, ch := range text[cookieStart:] {
		if ch == ';' {
			end = cookieStart + i
			break
		}
	}
	if end < 0 {
		return ""
	}
	return text[cookieStart:end]
}
