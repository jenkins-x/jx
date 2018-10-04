package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// CreateExtensionsOptions the flags for running create cluster
type UpdateExtensionsOptions struct {
	UpdateOptions
	InstallOptions InstallOptions
	Flags          InitFlags
	Provider       string
}

type UpdateExtensionsFlags struct {
}

var (
	updateExtensionsLong = templates.LongDesc(`
		This command updates extensions

`)

	updateExtensionsExample = templates.Examples(`

		jx update extensions repository

`)
)

func NewCmdUpdateExtensions(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpdateExtensionsOptions{
		UpdateOptions: UpdateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,

				Out: out,
				Err: errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "extensions",
		Short:   "Updates Extensions",
		Long:    fmt.Sprintf(updateExtensionsLong),
		Example: updateExtensionsExample,
		Run: func(cmd2 *cobra.Command, args []string) {
			options.Cmd = cmd2
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdUpdateExtensionsRepository(f, in, out, errOut))

	return cmd
}

func (o *UpdateExtensionsOptions) Run() error {
	return o.Cmd.Help()
}
