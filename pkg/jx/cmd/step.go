package cmd

import (
	"github.com/spf13/cobra"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepOptions struct {
	*CommonOptions

	DisableImport bool
	OutDir        string
}

var ()

// NewCmdStep Steps a command object for the "step" command
func NewCmdStep(commonOpts *CommonOptions) *cobra.Command {
	options := &StepOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "step",
		Short:   "pipeline steps",
		Aliases: []string{"steps"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepBuildPack(commonOpts))
	cmd.AddCommand(NewCmdStepBDD(commonOpts))
	cmd.AddCommand(NewCmdStepBlog(commonOpts))
	cmd.AddCommand(NewCmdStepChangelog(commonOpts))
	cmd.AddCommand(NewCmdStepCreate(commonOpts))
	cmd.AddCommand(NewCmdStepEnv(commonOpts))
	cmd.AddCommand(NewCmdStepGit(commonOpts))
	cmd.AddCommand(NewCmdStepGpgCredentials(commonOpts))
	cmd.AddCommand(NewCmdStepHelm(commonOpts))
	cmd.AddCommand(NewCmdStepLinkServices(commonOpts))
	cmd.AddCommand(NewCmdStepNexus(commonOpts))
	cmd.AddCommand(NewCmdStepNextVersion(commonOpts))
	cmd.AddCommand(NewCmdStepNextBuildNumber(commonOpts))
	cmd.AddCommand(NewCmdStepPre(commonOpts))
	cmd.AddCommand(NewCmdStepPR(commonOpts))
	cmd.AddCommand(NewCmdStepPost(commonOpts))
	cmd.AddCommand(NewCmdStepReport(commonOpts))
	cmd.AddCommand(NewCmdStepRelease(commonOpts))
	cmd.AddCommand(NewCmdStepSplitMonorepo(commonOpts))
	cmd.AddCommand(NewCmdStepTag(commonOpts))
	cmd.AddCommand(NewCmdStepValidate(commonOpts))
	cmd.AddCommand(NewCmdStepVerify(commonOpts))
	cmd.AddCommand(NewCmdStepWaitForArtifact(commonOpts))
	cmd.AddCommand(NewCmdStepCollect(commonOpts))

	return cmd
}

// Run implements this command
func (o *StepOptions) Run() error {
	return o.Cmd.Help()
}
