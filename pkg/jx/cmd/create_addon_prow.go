package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	defaultProwReleaseName = "prow"
	defaultProwNamespace   = "prow"
	prowVersion            = "0.0.3"
)

var (
	createAddonProwLong = templates.LongDesc(`
		Creates the prow addon for handling webhook events
`)

	createAddonProwExample = templates.Examples(`
		# Create the prow addon 
		jx create addon prow

		# Create the prow addon in a custom namespace
		jx create addon prow -n mynamespace
	`)
)

// CreateAddonProwOptions the options for the create spring command
type CreateAddonProwOptions struct {
	CreateAddonOptions

	Chart      string
	HMACToken  string
	OAUTHToken string
	Username   string
}

// NewCmdCreateAddonProw creates a command object for the "create" command
func NewCmdCreateAddonProw(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonProwOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "prow",
		Short:   "Create a prow addon",
		Long:    createAddonProwLong,
		Example: createAddonProwExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultProwNamespace, defaultProwReleaseName)

	cmd.Flags().StringVarP(&options.Version, "version", "v", prowVersion, "The version of the prow addon to use")
	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartProw, "The name of the chart to use")
	cmd.Flags().StringVarP(&options.HMACToken, "hmac-token", "", "", "OPTIONAL: The hmac-token is the token that you give to GitHub for validating webhooks. Generate it using any reasonable randomness-generator, eg openssl rand -hex 20")
	cmd.Flags().StringVarP(&options.OAUTHToken, "oauth-token", "", "", "OPTIONAL: The oauth-token is an OAuth2 token that has read and write access to the bot account. Generate it from the account's settings -> Personal access tokens -> Generate new token.")
	cmd.Flags().StringVarP(&options.Username, "username", "", "", "Overwrite cluster admin username")
	return cmd
}

// Run implements the command
func (o *CreateAddonProwOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}

	var err error
	if o.HMACToken == "" {
		o.HMACToken, err = util.RandStringBytesMaskImprSrc(41)
		if err != nil {
			return fmt.Errorf("cannot create a random hmac token for Prow")
		}
	}

	if o.OAUTHToken == "" {
		authConfigSvc, err := o.CreateGitAuthConfigService()
		if err != nil {
			return err
		}

		config := authConfigSvc.Config()
		server := config.GetOrCreateServer(config.CurrentServer)
		userAuth, err := config.PickServerUserAuth(server, "Git account to be used to send webhook events", o.BatchMode)
		if err != nil {
			return err
		}
		o.OAUTHToken = userAuth.ApiToken
	}

	if o.Username == "" {
		o.Username, err = o.GetClusterUserName()
		if err != nil {
			return err
		}
	}

	devNamespace, _, err := kube.GetDevNamespace(o.kubeClient, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	values := []string{"user=" + o.Username, "oauthToken=" + o.OAUTHToken, "hmacToken=" + o.HMACToken}
	err = o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values)
	if err != nil {
		return err
	}

	// create the ingress rule
	err = o.expose(devNamespace, o.Namespace, "")
	if err != nil {
		return err
	}
	return nil
}
