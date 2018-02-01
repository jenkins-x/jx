package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// StepNexusDropOptions contains the command line flags
type StepNexusDropOptions struct {
	StepNexusOptions
}

var (
	StepNexusDropLong = templates.LongDesc(`
		This pipeline step command drops a Staging Nexus Repository

`)

	StepNexusDropExample = templates.Examples(`
		jx step nexus drop

`)
)

func NewCmdStepNexusDrop(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepNexusDropOptions{
		StepNexusOptions: StepNexusOptions{
			StepOptions: StepOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "drop",
		Short:   "Drops a staging nexus release",
		Aliases: []string{"nexus_stage"},
		Long:    StepNexusDropLong,
		Example: StepNexusDropExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	//cmd.Flags().StringVarP(&options.Flags.Version, VERSION, "v", "", "version number for the tag [required]")
	return cmd
}

func (o *StepNexusDropOptions) Run() error {
	repoIds, err := o.findStagingRepoIds()
	if err != nil {
		return err
	}
	if len(repoIds) == 0 {
		o.Printf("No Nexus staging repository ids found in %s\n", util.ColorInfo(statingRepositoryProperties))
		return nil
	}
	return o.dropRepositories(repoIds, "Dropping staging repositories")
}
