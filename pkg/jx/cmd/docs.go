package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/pkg/browser"
)

const (
	docsURL = "https://jenkins-x.io/documentation/"
)

type DocsOptions struct {
	CommonOptions
}

func NewCmdDocs(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DocsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "docs",
		Short: "open the docs",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)

	return cmd
}

func (o *DocsOptions) Run() error {
	println("shizzle")
	browser.OpenURL(docsURL)
	return nil
}
