package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// StepNexusReleaseOptions contains the command line flags
type StepNexusReleaseOptions struct {
	StepNexusOptions

	DropOnFailure bool
}

var (
	StepNexusReleaseLong = templates.LongDesc(`
		This pipeline step command releases a Nexus staging repository
`)

	StepNexusReleaseExample = templates.Examples(`
		jx step nexus release

`)
)

func NewCmdStepNexusRelease(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepNexusReleaseOptions{
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
		Use:     "release",
		Short:   "Releases a staging nexus release",
		Aliases: []string{"nexus_stage"},
		Long:    StepNexusReleaseLong,
		Example: StepNexusReleaseExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.DropOnFailure, "drop-on-fail", "d", true, "Should we drop the repository on failure")
	return cmd
}

func (o *StepNexusReleaseOptions) Run() error {
	repoIds, err := o.findStagingRepoIds()
	if err != nil {
		return err
	}
	m := map[string]string{}

	if len(repoIds) == 0 {
		log.Infof("No Nexus staging repository ids found in %s\n", util.ColorInfo(statingRepositoryProperties))
		return nil
	}
	var answer error
	for _, repoId := range repoIds {
		err = o.releaseRepository(repoId)
		if err != nil {
			m[repoId] = fmt.Sprintf("Failed to release %s due to %s", repoId, err)
			if answer != nil {
				answer = err
			}
		}
	}
	if len(m) > 0 && o.DropOnFailure {
		for repoId, message := range m {
			err = o.dropRepository(repoId, message)
			if answer != nil {
				answer = err
			}
		}
	}
	return answer
}
