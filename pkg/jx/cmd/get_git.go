package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// GetGitOptions the command line options
type GetGitOptions struct {
	GetOptions
}

var (
	get_git_long = templates.LongDesc(`
		Display the Git server URLs.

`)

	get_git_example = templates.Examples(`
		# List all registered Git server URLs
		jx get git
	`)
)

// NewCmdGetGit creates the command
func NewCmdGetGit(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetGitOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "git [flags]",
		Short:   "Display the current registered Git service URLs",
		Long:    get_git_long,
		Example: get_git_example,
		Aliases: []string{"gitserver"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	return cmd
}

// Run implements this command
func (o *GetGitOptions) Run() error {
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	table := o.createTable()
	table.AddRow("Name", "Kind", "URL")

	for _, s := range config.Servers {
		kind := s.Kind
		if kind == "" {
			kind = "github"
		}
		table.AddRow(s.Name, kind, s.URL)
	}
	table.Render()
	return nil
}
