package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chromedp/chromedp/runner"
	"github.com/hpcloud/tail"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	ssoCookieName      = "sso-cb-cdx"
	onboardingEndpoint = "/api/v1/users/loggedin"
)

type LoginOptions struct {
	CommonOptions

	URL string
}

var (
	login_long = templates.LongDesc(`
		Onboards an user into the CloudBees application and configures the Kubernetes client configuration.

		A CloudBess app can be created as an addon with 'jx create addon cloudbees'`)

	login_example = templates.Examples(`
		# Onboard into CloudBees application
		jx login`)
)

func NewCmdLogin(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &LoginOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
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

	return cmd
}

func (o *LoginOptions) Run() error {
	cookie, err := o.Login()
	if err != nil {
		return errors.Wrap(err, "loging into the CloudBees application")
	}
	if cookie == "" {
		return errors.New("failed to log into the CloudBees application")
	}

	resp, err := o.onboardUser(cookie)
	if err != nil {
		return errors.Wrap(err, "onboarding user")
	}

	fmt.Println(*resp)
	return nil
}

func (o *LoginOptions) Login() (string, error) {
	url := o.URL
	if url == "" {
		return "", errors.New("please povide the URL of the CloudBees application in '--url' option")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	userDataDir, err := ioutil.TempDir("/tmp", "jx-login-chrome-userdata-dir")
	if err != nil {
		return "", errors.Wrap(err, "creating the chrome user data dir")
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
		return "", errors.Wrap(err, "creating chrome runner")
	}

	err = r.Start(ctx)
	if err != nil {
		return "", errors.Wrap(err, "starting chrome")
	}

	t, err := tail.TailFile(netLogFile, tail.Config{
		Follow: true,
		Logger: log.New(ioutil.Discard, "", log.LstdFlags)})
	if err != nil {
		return "", errors.Wrap(err, "reading the netlog file")
	}
	cookie := ""
	pattern := fmt.Sprintf("%s=", ssoCookieName)
	for line := range t.Lines {
		if strings.Contains(line.Text, pattern) {
			fmt.Println(line.Text)
			cookie = ExtractSsoCookie(line.Text)
			break
		}
	}

	err = r.Shutdown(ctx)
	if err != nil {
		return "", errors.Wrap(err, "shutting down Chrome")
	}

	err = r.Wait()
	if err != nil {
		return "", errors.Wrap(err, "waiting for Chrome  to exit")
	}

	return cookie, nil
}

func (o *LoginOptions) onboardUser(cookie string) (*string, error) {
	client := http.Client{}
	req, err := http.NewRequest("GET", o.onboardingURL(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "building onboarding request")
	}
	req.Header.Add("Accept", "application/json")
	ssoCookie := http.Cookie{Name: ssoCookieName, Value: cookie}
	req.AddCookie(&ssoCookie)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "executing onboarding request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("user onboarding status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading user onboarding information from response body")
	}

	userInfo := string(body)
	return &userInfo, nil
}

func (o *LoginOptions) onboardingURL() string {
	url := o.URL
	if strings.HasPrefix(url, "/") {
		url = strings.TrimPrefix(url, "/")
	}
	return url + onboardingEndpoint
}

func ExtractSsoCookie(text string) string {
	cookiePattern := fmt.Sprintf("%s=", ssoCookieName)
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
