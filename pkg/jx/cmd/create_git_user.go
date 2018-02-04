package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
	"k8s.io/apimachinery/pkg/util/uuid"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"bytes"
	"strings"
)

var (
	create_git_user_long = templates.LongDesc(`
		Adds a new Git Server URL
        ` + env_description + `
`)

	create_git_user_example = templates.Examples(`
		# Add a new git server URL
		jx create git server gitea mythingy https://my.server.com/
	`)
)

// CreateGitUserOptions the command line options for the command
type CreateGitUserOptions struct {
	CreateOptions

	GitServerFlags GitServerFlags
	Username       string
	Password       string
	ApiToken       string
}

// NewCmdCreateGitUser creates a command
func NewCmdCreateGitUser(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateGitUserOptions{
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
		Short:   "Adds a new user name and api token for a git server server",
		Aliases: []string{"token"},
		Long:    create_git_user_long,
		Example: create_git_user_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	options.GitServerFlags.addGitServerFlags(cmd)
	cmd.Flags().StringVarP(&options.ApiToken, "api-token", "t", "", "The API Token for the user")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "The User password to try automatically create a new API Token")

	return cmd
}

// Run implements the command
func (o *CreateGitUserOptions) Run() error {
	args := o.Args
	if len(args) > 0 {
		o.Username = args[0]
	}
	if len(args) > 1 {
		o.ApiToken = args[1]
	}
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	server, err := o.findGitServer(config, &o.GitServerFlags)
	if err != nil {
		return err
	}

	// TODO add the API thingy...
	if o.Username == "" {
		return fmt.Errorf("No Username specified")
	}

	userAuth := config.GetOrCreateUserAuth(server.URL, o.Username)
	if o.ApiToken != "" {
		userAuth.ApiToken = o.ApiToken
	}

	tokenUrl := gits.ProviderAccessTokenURL(server.Kind, server.URL)

	if userAuth.IsInvalid() && o.Password != "" {
		err := o.tryFindAPITokenFromBrowser(tokenUrl, userAuth)
		if err != nil {
			return err
		}
	}

	if userAuth.IsInvalid() {
		o.Printf("Please generate an API Token for server %s\n", server.Label())
		o.Printf("Click this URL %s\n\n", util.ColorInfo(tokenUrl))
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
func (o *CreateGitUserOptions) tryFindAPITokenFromBrowser(tokenUrl string, userAuth *auth.UserAuth) error {
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

	nodes := []*cdp.Node{}
	err = c.Run(ctxt, chromedp.Nodes("//input", &nodes))
	if err != nil {
		return err
	}

	login := false
	for _, node := range nodes {
		name := node.AttributeValue("name")
		if name == "user_name" {
			login = true
		}
	}

	if login {
		o.Printf("logging in\n")
		err = c.Run(ctxt, chromedp.Tasks{
			chromedp.WaitVisible("user_name", chromedp.ByID),
			chromedp.SendKeys("user_name", userAuth.Username, chromedp.ByID),
			chromedp.SendKeys("password", o.Password+"\n", chromedp.ByID),
		})
		if err != nil {
			return err
		}
	}
	o.Printf("Generating new token\n")

	tokenId := "jx-" + string(uuid.NewUUID())
	generateNewTokenButtonSelector := "//div[normalize-space(text())='Generate New Token']"

	tokenResultDivSelector := "//div[@class='ui info message']/p"
	err = c.Run(ctxt, chromedp.Tasks{
		chromedp.WaitVisible(generateNewTokenButtonSelector),
		chromedp.Click(generateNewTokenButtonSelector),
		chromedp.WaitVisible("name", chromedp.ByID),
		chromedp.SendKeys("name", tokenId + "\n", chromedp.ByID),
		chromedp.WaitVisible(tokenResultDivSelector),
		chromedp.Nodes(tokenResultDivSelector, &nodes),
	})
	if err != nil {
		return err
	}
	token := ""
	for _, node := range nodes {
		text := nodeText(node)
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

func nodeText(node *cdp.Node) string {
	var buffer bytes.Buffer
	for _, node := range node.Children {
		switch node.NodeType {
		case cdp.NodeTypeText:
			buffer.WriteString(node.NodeValue)
		case cdp.NodeTypeElement:
			buffer.WriteString(nodeText(node))
		}
	}
	return strings.TrimSpace(buffer.String())
}
