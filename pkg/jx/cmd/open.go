package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

type OpenOptions struct {
	ConsoleOptions
}

var (
	open_long = templates.LongDesc(`
		Opens a named service in the browser.

		You can use the '--url' argument to just display the URL without opening it`)

	open_example = templates.Examples(`
		# Open the Nexus console in a browser
		jx open jenkins-x-sonatype-nexus

		# Print the Nexus console URL but do not open a browser
		jx open jenkins-x-sonatype-nexus -u

		# List all the service URLs
		jx open`)
)

func NewCmdOpen(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &OpenOptions{
		ConsoleOptions: ConsoleOptions{
			GetURLOptions: GetURLOptions{
				GetOptions: GetOptions{
					CommonOptions: commoncmd.CommonOptions{
						Factory: f,
						In:      in,

						Out: out,
						Err: errOut,
					},
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "open",
		Short:   "Open a service in a browser",
		Long:    open_long,
		Example: open_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addConsoleFlags(cmd)
	return cmd
}

func (o *OpenOptions) Run() error {
	if len(o.Args) == 0 {
		return o.GetURLOptions.Run()
	}
	name := o.Args[0]
	return o.ConsoleOptions.Open(name, name)
}
