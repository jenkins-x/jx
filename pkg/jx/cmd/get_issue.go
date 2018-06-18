package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetIssueOptions contains the command line options
type GetIssueOptions struct {
	GetOptions

	Dir string
	Id  int32
}

var (
	GetIssueLong = templates.LongDesc(`
		Display the status of an issue for a project.

`)

	GetIssueExample = templates.Examples(`
		# Get the status of an issue for a project
		jx get issue --id ISSUE_ID
	`)
)

// NewCmdGetIssue creates the command
func NewCmdGetIssue(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetIssueOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "issue [flags]",
		Short:   "Display the status of an issue",
		Long:    GetIssueLong,
		Example: GetIssueExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().Int32VarP(&options.Id, "id", "i", 0, "The issue ID")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The root project directory")

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetIssueOptions) Run() error {
	return nil
}
