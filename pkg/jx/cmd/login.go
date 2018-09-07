package cmd

import (
	"context"
	"io"

	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
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
	return o.login()
}

func (o *LoginOptions) login() error {
	url := o.URL
	if url == "" {
		return errors.New("please povide the URL of the CloudBees application in '--url' option")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chrome, err := chromedp.New(ctx)
	if err != nil {
		return errors.Wrap(err, "creating chrome client")
	}

	err = chrome.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(url),
	})
	if err != nil {
		return errors.Wrap(err, "open chrome with CloudBees application URL")
	}
	return chrome.Wait()
}
