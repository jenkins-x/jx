package cmd

import (
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// GetIssuesOptions contains the command line options
type GetIssuesOptions struct {
	GetOptions
	Dir    string
	Filter string
}

var (
	GetIssuesLong = templates.LongDesc(`
		Display one or more issues for a project.

`)

	GetIssuesExample = templates.Examples(`
		# List open issues on the current project
		jx get issues
	`)
)

// NewCmdGetIssues creates the command
func NewCmdGetIssues(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetIssuesOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,

				Out: out,
				Err: errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "issues [flags]",
		Short:   "Display one or more issues",
		Long:    GetIssuesLong,
		Example: GetIssuesExample,
		Aliases: []string{"jira"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The root project directory")

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetIssuesOptions) Run() error {
	tracker, err := o.createIssueProvider(o.Dir)
	if err != nil {
		return err
	}

	issues, err := tracker.SearchIssues(o.Filter)
	if err != nil {
		return err
	}

	table := o.createTable()
	table.AddRow("ISSUE", "TITLE")
	for _, i := range issues {
		table.AddRow(i.URL, i.Title)
	}
	table.Render()
	return nil
}

func (o *GetIssuesOptions) matchesFilter(job *gojenkins.Job) bool {
	args := o.Args
	if len(args) == 0 {
		return true
	}
	name := job.FullName
	for _, arg := range args {
		if strings.Contains(name, arg) {
			return true
		}
	}
	return false
}
