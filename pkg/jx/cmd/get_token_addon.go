package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetTokenAddonOptions the command line options
type GetTokenAddonOptions struct {
	GetOptions

	Kind string
	Name string
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
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
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
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "", "Filters the addonss by the kind")
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
		o.Printf("No addon servers registered. To register a new token for an addon server use: %s\n", util.ColorInfo("jx create token addon"))
		return nil
	}

	filterKind := o.Kind
	filterName := o.Name

	table := o.CreateTable()
	table.AddRow("KIND", "NAME", "URL", "USERNAME", "TOKEN?")

	for _, s := range config.Servers {
		kind := s.Kind
		name := s.Name
		if (filterKind == "" || filterKind == kind) && (filterName == "" || filterName == name) {
			user := ""
			pwd := ""
			if len(s.Users) == 0 {
				table.AddRow(kind, name, kind, s.URL, user, pwd)
			} else {
				for _, u := range s.Users {
					user = u.Username
					pwd = ""
					if u.ApiToken != "" {
						pwd = "yes"
					}
				}
				table.AddRow(kind, name, s.URL, user, pwd)
			}
		}
	}
	table.Render()
	return nil
}
