package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetTokenOptions the command line options
type GetTokenOptions struct {
	GetOptions

	Kind string
	Name string
}

// NewCmdGetToken creates the command
func NewCmdGetToken(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetTokenOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "token",
		Short:   "Display the tokens for different kinds of services",
		Aliases: []string{"api-token"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdGetTokenAddon(f, out, errOut))
	return cmd
}

func (o *GetTokenOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Kind, "kind", "k", "", "Filters the services by the kind")
	cmd.Flags().StringVarP(&o.Name, "name", "n", "", "Filters the services by the name")
}

// Run implements this command
func (o *GetTokenOptions) Run() error {
	return o.Cmd.Help()
}

func (o *GetTokenOptions) displayUsersWithTokens(authConfigSvc auth.AuthConfigService) error {
	config := authConfigSvc.Config()

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
