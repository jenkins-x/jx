package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetTokenAddonOptions the command line options
type GetTokenAddonOptions struct {
	GetTokenOptions
}

var (
	getTokenAddonLong = templates.LongDesc(`
		Display the users with tokens for the addons

`)

	getTokenAddonExample = templates.Examples(`
		# List all users with tokens for all addons
		jx get token addon
	`)
)

// NewCmdGetTokenAddon creates the command
func NewCmdGetTokenAddon(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetTokenAddonOptions{
		GetTokenOptions{
			GetOptions: GetOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "addon",
		Short:   "Display the current users and if they have a token for the addons",
		Long:    getTokenAddonLong,
		Example: getTokenAddonExample,
		Aliases: []string{"issue-tracker"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetTokenAddonOptions) Run() error {
	authConfigSvc, err := o.CreateAddonAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()
	if len(config.Servers) == 0 {
		log.Warnf("No addon servers registered. To register a new token for an addon server use: %s\n", util.ColorInfo("jx create token addon"))
		return nil
	}
	return o.displayUsersWithTokens(authConfigSvc)
}
