package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	deleteAddonSSOLong = templates.LongDesc(`
		Deletes the SSO addon
`)

	deleteAddonSSOExample = templates.Examples(`
		# Deletes the SSO addon
		jx delete addon sso
	`)
)

// DeleteAddonSSOOptions the options for delete addon sso command
type DeleteAddonSSOOptions struct {
	DeleteAddonOptions
}

// NewCmdDeleteAddonSSO defines the command
func NewCmdDeleteAddonSSO(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteAddonSSOOptions{
		DeleteAddonOptions: DeleteAddonOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "sso",
		Short:   "Deletes the Single Sing-On addon",
		Aliases: []string{"sso"},
		Long:    deleteAddonSSOLong,
		Example: deleteAddonSSOExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteAddonSSOOptions) Run() error {
	return nil
}
