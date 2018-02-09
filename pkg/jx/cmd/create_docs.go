package cmd

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

var (
	create_docs_long = templates.LongDesc(`
		Creates the documentation markdown files
`)

	create_docs_example = templates.Examples(`
		# Create the documentation files
		jx create docs
	`)
)

// CreateDocsOptions the options for the create spring command
type CreateDocsOptions struct {
	CreateOptions

}

// NewCmdCreateDocs creates a command object for the "create" command
func NewCmdCreateDocs(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateDocsOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "docs",
		Short:   "Creates the documentation files",
		Aliases: []string{"doc"},
		Long:    create_docs_long,
		Example: create_docs_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements the command
func (o *CreateDocsOptions) Run() error {
	jxcommand := NewJXCommand(o.Factory, os.Stdin, ioutil.Discard, ioutil.Discard)
	return doc.GenMarkdownTree(jxcommand, "./")
	return nil
}
