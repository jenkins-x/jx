package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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
func NewCmdStep(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
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
			CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdStepBuildPack(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepBDD(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepBlog(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepChangelog(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepCredential(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepCreate(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepEnv(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepGet(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepGit(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepGpgCredentials(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepHelm(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepLinkServices(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepNexus(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepNextVersion(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepNextBuildNumber(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepPre(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepPR(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepPost(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepRelease(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepSplitMonorepo(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepTag(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepValidate(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepVerify(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepWaitForArtifact(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepStash(f, in, out, errOut))
	cmd.AddCommand(NewCmdStepUnstash(f, in, out, errOut))

	return cmd
}

// Run implements this command
func (o *StepOptions) Run() error {
	return o.Cmd.Help()
}
