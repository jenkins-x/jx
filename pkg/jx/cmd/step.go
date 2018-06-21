package cmd

import (
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepOptions struct {
	CommonOptions

	DisableImport bool
	OutDir        string
}

var ()

// NewCmdStep Steps a command object for the "step" command
func NewCmdStep(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "step",
		Short:   "pipeline steps",
		Aliases: []string{"steps"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepBlog(f, out, errOut))
	cmd.AddCommand(NewCmdStepChangelog(f, out, errOut))
	cmd.AddCommand(NewCmdStepEnvRoleBinding(f, out, errOut))
	cmd.AddCommand(NewCmdStepGit(f, out, errOut))
	cmd.AddCommand(NewCmdStepGpgCredentials(f, out, errOut))
	cmd.AddCommand(NewCmdStepHelm(f, out, errOut))
	cmd.AddCommand(NewCmdStepNexus(f, out, errOut))
	cmd.AddCommand(NewCmdStepNextVersion(f, out, errOut))
	cmd.AddCommand(NewCmdStepPR(f, out, errOut))
	cmd.AddCommand(NewCmdStepPost(f, out, errOut))
	cmd.AddCommand(NewCmdStepReport(f, out, errOut))
	cmd.AddCommand(NewCmdStepSplitMonorepo(f, out, errOut))
	cmd.AddCommand(NewCmdStepTag(f, out, errOut))
	cmd.AddCommand(NewCmdStepValidate(f, out, errOut))
	cmd.AddCommand(NewCmdStepWaitForArtifact(f, out, errOut))

	return cmd
}

// Run implements this command
func (o *StepOptions) Run() error {
	return o.Cmd.Help()
}
