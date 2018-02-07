package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetGitOptions the command line options
type GetGitOptions struct {
	GetOptions
}

var (
	get_git_long = templates.LongDesc(`
		Display the git server URLs.

`)

	get_git_example = templates.Examples(`
		# List all registered git server URLs
		jx get git
	`)
)

// NewCmdGetGit creates the command
func NewCmdGetGit(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetGitOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "git [flags]",
		Short:   "Display the current registered git service URLs",
		Long:    get_git_long,
		Example: get_git_example,
		Aliases: []string{"gitserver"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	return cmd
}

// Run implements this command
func (o *GetGitOptions) Run() error {
	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config := authConfigSvc.Config()

	table := o.CreateTable()
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
