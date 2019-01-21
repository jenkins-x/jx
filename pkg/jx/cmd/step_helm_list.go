package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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

func NewCmdStepHelmList(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepHelmListOptions{
		StepHelmOptions: StepHelmOptions{
			StepOptions: StepOptions{
				CommonOptions: commoncmd.CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
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
	log.Info(output)
	return nil
}
