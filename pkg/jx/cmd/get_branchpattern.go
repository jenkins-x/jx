package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetBranchPatternOptions containers the CLI options
type GetBranchPatternOptions struct {
	GetOptions
}

const (
	branchPattern = "branchpattern"

	defaultBranchPatterns     = jenkins.BranchPatternMasterPRsAndFeatures
	defaultForkBranchPatterns = ""
)

var (
	branchPatternsAliases = []string{
		"branch pattern",
	}

	getBranchPatternLong = templates.LongDesc(`
		Display the git branch patterns for the current Team used on creating and importing projects

		For more documentation see: [https://jenkins-x.io/developing/import/#branch-patterns](https://jenkins-x.io/developing/import/#branch-patterns)
`)

	getBranchPatternExample = templates.Examples(`
		# List the git branch patterns for the current team
		jx get branchpattern
	`)
)

// NewCmdGetBranchPattern creates the new command for: jx get env
func NewCmdGetBranchPattern(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetBranchPatternOptions{
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
		Use:     branchPattern,
		Short:   "Display the git branch patterns for the current Team used on creating and importing projects",
		Aliases: branchPatternsAliases,
		Long:    getBranchPatternLong,
		Example: getBranchPatternExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetBranchPatternOptions) Run() error {
	patterns, err := o.TeamBranchPatterns()
	if err != nil {
		return err
	}
	table := o.createTable()
	table.AddRow("BRANCH PATTERNS")
	table.AddRow(patterns.DefaultBranchPattern)
	table.Render()
	return nil
}
