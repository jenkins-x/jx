package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
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
		Use:     "user [username]",
		Short:   "Adds a new user name and api token for a jenkins server server",
		Aliases: []string{"token"},
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
	authConfigSvc, err := o.Factory.CreateJenkinsAuthConfigService()
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
		server, err = o.findServer(config, &o.ServerFlags, "jenkins server", "Try installing one via: jx create team")
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

	tokenUrl := jenkins.JenkinsTokenURL(server.URL)

	if userAuth.IsInvalid() && o.Password != "" {
		err := o.tryFindAPITokenFromBrowser(tokenUrl, userAuth)
		if err != nil {
			return err
		}
	}

	if userAuth.IsInvalid() {
		jenkins.PrintGetTokenFromURL(o.Out, tokenUrl)
		o.Printf("Then COPY the token and enter in into the form below:\n\n")

		err = config.EditUserAuth(userAuth, o.Username, false)
		if err != nil {
			return err
		}
		if userAuth.IsInvalid() {
			return fmt.Errorf("You did not properly define the user authentication!")
		}
	}

	err = authConfigSvc.SaveConfig()
	if err != nil {
		return err
	}
	o.Printf("Created user %s API Token for git server %s at %s\n",
		util.ColorInfo(o.Username), util.ColorInfo(server.Name), util.ColorInfo(server.URL))
	return nil
}

// lets try use the users browser to find the API token
func (o *CreateJenkinsUserOptions) tryFindAPITokenFromBrowser(tokenUrl string, userAuth *auth.UserAuth) error {
	var err error

	ctxt, cancel := context.WithCancel(context.Background())
	defer cancel()

	file, err := ioutil.TempFile("", "jx-browser")
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	o.Printf("Chrome debugging logs written to: %s\n", util.ColorInfo(file.Name()))

	logger := func(message string, args ...interface{}) {
		fmt.Fprintf(writer, message+"\n", args...)
	}
	c, err := chromedp.New(ctxt, chromedp.WithLog(logger))
	if err != nil {
		log.Fatal(err)
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
		o.Printf("logging in\n")
		err = c.Run(ctxt, chromedp.Tasks{
			chromedp.WaitVisible(userNameInputName, chromedp.ByID),
			chromedp.SendKeys(userNameInputName, userAuth.Username, chromedp.ByID),
			chromedp.SendKeys(passwordInputSelector, o.Password+"\n"),
		})
		if err != nil {
			return err
		}
	}
	o.Printf("Getting the API Token...\n")

	getAPITokenButtonSelector := "//button[normalize-space(text())='Show API Token...']"
	//tokenInputSelector := "//input[@name='_.apiToken']"
	nodeSlice = []*cdp.Node{}
	err = c.Run(ctxt, chromedp.Tasks{
		chromedp.WaitVisible(getAPITokenButtonSelector),
		chromedp.Click(getAPITokenButtonSelector),
		chromedp.WaitVisible("apiToken", chromedp.ByID),
		chromedp.Nodes("apiToken", &nodeSlice, chromedp.ByID),
	})
	if err != nil {
		return err
	}
	token := ""
	o.Printf("Got Nodes %#v\n", nodeSlice)
	for _, node := range nodeSlice {
		text := node.AttributeValue("value")
		if text != "" && token == "" {
			token = text
			break
		}
	}
	o.Printf("Found API Token %s\n", util.ColorInfo(token))
	if token != "" {
		userAuth.ApiToken = token
	}

	err = c.Shutdown(ctxt)
	if err != nil {
		return err
	}
	return nil
}
