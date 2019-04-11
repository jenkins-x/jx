package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// StepHelmListOptions contains the command line flags
type StepHelmListOptions struct {
	StepHelmOptions

	Namespace string
}

var (
	StepHelmListLong = templates.LongDesc(`
		List the helm releases
`)

	StepHelmListExample = templates.Examples(`
		# list all the helm releases in the current namespace
		jx step helm list

`)
)

func NewCmdStepHelmList(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepHelmListOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List the helm releases",
		Aliases: []string{""},
		Long:    StepHelmListLong,
		Example: StepHelmListExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addStepHelmFlags(cmd)
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to look for the helm releases. Defaults to the current namespace")

	return cmd
}

func (o *StepHelmListOptions) Run() error {
	h := o.Helm()
	if h == nil {
		return fmt.Errorf("No Helmer created!")
	}
	output, err := h.ListCharts()
	if err != nil {
		return err
	}
	logrus.Info(output)
	return nil
}
