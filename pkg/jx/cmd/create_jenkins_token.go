package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/runner"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	create_jenkins_user_long = templates.LongDesc(`
		Creates a new user and API Token for the current Jenkins Server
`)

	create_jenkins_user_example = templates.Examples(`
		# Add a new API Token for a user for the current Jenkins server
        # prompting the user to find and enter the API Token
		jx create jenkins token someUserName

		# Add a new API Token for a user for the current Jenkins server
 		# using browser automation to login to the git server
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
	Timeout     string
	UseBrowser  bool
}

// NewCmdCreateJenkinsUser creates a command
func NewCmdCreateJenkinsUser(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateJenkinsUserOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "token [username]",
		Short:   "Adds a new username and api token for a Jenkins server",
		Aliases: []string{"api-token"},
		Long:    create_jenkins_user_long,
		Example: create_jenkins_user_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	options.ServerFlags.addGitServerFlags(cmd)
	cmd.Flags().StringVarP(&options.ApiToken, "api-token", "t", "", "The API Token for the user")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "The User password to try automatically create a new API Token")
	cmd.Flags().StringVarP(&options.Timeout, "timeout", "", "", "The timeout if using browser automation to generate the API token (by passing username and password)")
	cmd.Flags().BoolVarP(&options.UseBrowser, "browser", "", false, "Use a Chrome browser to automatically find the API token if the user and password are known")

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
	kubeClient, ns, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("error connecting to kubernetes cluster: %v", err)
	}

	authConfigSvc, err := o.Factory.CreateJenkinsAuthConfigService(kubeClient, ns)
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

	if o.Password != "" {
		userAuth.Password = o.Password
	}

	tokenUrl := jenkins.JenkinsTokenURL(server.URL)
	if o.Verbose {
		log.Infof("using url %s\n", tokenUrl)
	}
	if userAuth.IsInvalid() && o.Password != "" && o.UseBrowser {
		err := o.tryFindAPITokenFromBrowser(tokenUrl, userAuth)
		if err != nil {
			log.Warnf("unable to automatically find API token with chromedp using URL %s\n", tokenUrl)
		}
	}

	if userAuth.IsInvalid() {
		f := func(username string) error {
			jenkins.PrintGetTokenFromURL(o.Out, tokenUrl)
			log.Infof("Then COPY the token and enter in into the form below:\n\n")
			return nil
		}

		err = config.EditUserAuth("Jenkins", userAuth, o.Username, false, o.BatchMode, f)
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
	s, err := o.kubeClient.CoreV1().Secrets(o.currentNamespace).Get(kube.SecretJenkins, metav1.GetOptions{})
	if err != nil {
		return err
	}
	s.Data[kube.JenkinsAdminApiToken] = []byte(userAuth.ApiToken)

	_, err = o.kubeClient.CoreV1().Secrets(o.currentNamespace).Update(s)
	if err != nil {
		return err
	}

	log.Infof("Created user %s API Token for Jenkins server %s at %s\n",
		util.ColorInfo(o.Username), util.ColorInfo(server.Name), util.ColorInfo(server.URL))
	return nil
}

// lets try use the users browser to find the API token
func (o *CreateJenkinsUserOptions) tryFindAPITokenFromBrowser(tokenUrl string, userAuth *auth.UserAuth) error {
	var ctxt context.Context
	var cancel context.CancelFunc
	if o.Timeout != "" {
		duration, err := time.ParseDuration(o.Timeout)
		if err != nil {
			return err
		}
		ctxt, cancel = context.WithTimeout(context.Background(), duration)
	} else {
		ctxt, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	c, err := o.createChromeClient(ctxt)
	if err != nil {
		return err
	}

	err = c.Run(ctxt, chromedp.Tasks{
		chromedp.Navigate(tokenUrl),
	})
	if err != nil {
		return err
	}

	nodeSlice := []*cdp.Node{}
	err = c.Run(ctxt, chromedp.Nodes("//input", &nodeSlice))
	if err != nil {
		return err
	}

	login := false
	userNameInputName := "j_username"
	passwordInputSelector := "//input[@name='j_password']"
	for _, node := range nodeSlice {
		name := node.AttributeValue("name")
		if name == userNameInputName {
			login = true
		}
	}

	if login {
		// disable screenshots to try and reduce errors when running headless
		//o.captureScreenshot(ctxt, c, "screenshot-jenkins-login.png", "main-panel", chromedp.ByID)

		log.Infoln("logging in")
		err = c.Run(ctxt, chromedp.Tasks{
			chromedp.WaitVisible(userNameInputName, chromedp.ByID),
			chromedp.SendKeys(userNameInputName, userAuth.Username, chromedp.ByID),
			chromedp.SendKeys(passwordInputSelector, o.Password+"\n"),
		})
		if err != nil {
			return err
		}
	}

	// disable screenshots to try and reduce errors when running headless
	//o.captureScreenshot(ctxt, c, "screenshot-jenkins-api-token.png", "main-panel", chromedp.ByID)

	getAPITokenButtonSelector := "//button[normalize-space(text())='Show API Token...']"
	nodeSlice = []*cdp.Node{}

	log.Infoln("Getting the API Token...")
	err = c.Run(ctxt, chromedp.Tasks{
		chromedp.Sleep(2 * time.Second),
		chromedp.WaitVisible(getAPITokenButtonSelector),
		chromedp.Click(getAPITokenButtonSelector),
		//chromedp.WaitVisible("apiToken", chromedp.ByID),
		chromedp.Nodes("apiToken", &nodeSlice, chromedp.ByID),
	})
	if err != nil {
		return err
	}
	token := ""
	for _, node := range nodeSlice {
		text := node.AttributeValue("value")
		if text != "" && token == "" {
			token = text
			break
		}
	}
	log.Infoln("Found API Token")
	if token != "" {
		userAuth.ApiToken = token
	}

	err = c.Shutdown(ctxt)
	if err != nil {
		return err
	}
	return nil
}

// lets try use the users browser to find the API token
func (o *CommonOptions) createChromeClient(ctxt context.Context) (*chromedp.CDP, error) {
	if o.Headless {
		options := func(m map[string]interface{}) error {
			m["remote-debugging-port"] = 9222
			m["no-sandbox"] = true
			m["headless"] = true
			return nil
		}

		return chromedp.New(ctxt, chromedp.WithRunnerOptions(runner.CommandLineOption(options)))
	}
	return chromedp.New(ctxt)
}

func (o *CommonOptions) captureScreenshot(ctxt context.Context, c *chromedp.CDP, screenshotFile string, selector interface{}, options ...chromedp.QueryOption) error {
	log.Infoln("Creating a screenshot...")

	var picture []byte
	err := c.Run(ctxt, chromedp.Tasks{
		chromedp.Sleep(2 * time.Second),
		chromedp.Screenshot(selector, &picture, options...),
	})
	if err != nil {
		return err
	}
	log.Infoln("Saving a screenshot...")

	err = ioutil.WriteFile(screenshotFile, picture, util.DefaultWritePermissions)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Infof("Saved screenshot: %s\n", util.ColorInfo(screenshotFile))
	return err
}

func (o *CommonOptions) createChromeDPLogger() (chromedp.LogFunc, error) {
	var logger chromedp.LogFunc
	if o.Verbose {
		logger = func(message string, args ...interface{}) {
			log.Infof(message+"\n", args...)
		}
	} else {
		file, err := ioutil.TempFile("", "jx-browser")
		if err != nil {
			return logger, err
		}
		writer := bufio.NewWriter(file)
		log.Infof("Chrome debugging logs written to: %s\n", util.ColorInfo(file.Name()))

		logger = func(message string, args ...interface{}) {
			fmt.Fprintf(writer, message+"\n", args...)
		}
	}
	return logger, nil
}
