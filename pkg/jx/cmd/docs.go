package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/pkg/browser"
)

const (
	docsURL = "https://jenkins-x.io/documentation/"
)

/* open the docs - Jenkins X docs by default */
func NewCmdDocs(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Open the documentation in a browser",
		Run: func(cmd *cobra.Command, args []string) {
			err := browser.OpenURL(docsURL)
			CheckErr(err)
		},
	}
	return cmd
}
