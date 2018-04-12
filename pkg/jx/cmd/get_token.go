package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetTokenOptions the command line options
type GetTokenOptions struct {
	GetOptions
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

// Run implements this command
func (o *GetTokenOptions) Run() error {
	return o.Cmd.Help()
}
