package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/spf13/cobra"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepOptions struct {
	*opts.CommonOptions

	DisableImport bool
	OutDir        string
}

// NewCmdStep Steps a command object for the "step" command
func NewCmdStep(commonOpts *opts.CommonOptions) *cobra.Command {
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
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepBuildPack(commonOpts))
	cmd.AddCommand(NewCmdStepBDD(commonOpts))
	cmd.AddCommand(NewCmdStepBlog(commonOpts))
	cmd.AddCommand(NewCmdStepChangelog(commonOpts))
	cmd.AddCommand(NewCmdStepCredential(commonOpts))
	cmd.AddCommand(NewCmdStepCreate(commonOpts))
	cmd.AddCommand(NewCmdStepCustomPipeline(commonOpts))
	cmd.AddCommand(NewCmdStepEnv(commonOpts))
	cmd.AddCommand(NewCmdStepGet(commonOpts))
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
	cmd.AddCommand(NewCmdStepRelease(commonOpts))
	cmd.AddCommand(NewCmdStepSplitMonorepo(commonOpts))
	cmd.AddCommand(NewCmdStepSyntax(commonOpts))
	cmd.AddCommand(NewCmdStepTag(commonOpts))
	cmd.AddCommand(NewCmdStepValidate(commonOpts))
	cmd.AddCommand(NewCmdStepVerify(commonOpts))
	cmd.AddCommand(NewCmdStepWaitForArtifact(commonOpts))
	cmd.AddCommand(NewCmdStepStash(commonOpts))
	cmd.AddCommand(NewCmdStepUnstash(commonOpts))
	cmd.AddCommand(NewCmdStepValuesSchemaTemplate(commonOpts))
	cmd.AddCommand(NewCmdStepScheduler(commonOpts))

	return cmd
}

// Run implements this command
func (o *StepOptions) Run() error {
	return o.Cmd.Help()
}
