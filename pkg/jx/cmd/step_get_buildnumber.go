package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
)

var (
	getBuildNumberLong = templates.LongDesc(`
		Outputs the current build number from environment variables or using the Downward API inside build pods
`)

	getBuildNumberExample = templates.Examples(`
		# dislay the current build number
		jx step get buildnumber

			`)
)

// StepGetBuildNumberOptions contains the command line flags
type StepGetBuildNumberOptions struct {
	StepOptions

	Dir string
}

// NewCmdStepGetBuildNumber Creates a new Command object
func NewCmdStepGetBuildNumber(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepGetBuildNumberOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "buildnumber",
		Short:   "Outputs the current build number from environment variables or using the Downward API inside build pods",
		Long:    getBuildNumberLong,
		Example: getBuildNumberExample,
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
func (o *StepGetBuildNumberOptions) Run() error {
	text := o.GetBuildNumber()
	log.Infof("%s\n", text)
	return nil
}
