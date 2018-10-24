package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/prow"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// StartComplianceOptions contains the command line options
type StartComplianceOptions struct {
	GetOptions

	Tail   bool
	Filter string

	Jobs map[string]gojenkins.Job
}

var (
	start_compliance_long = templates.LongDesc(`
		Starts compliance checking on an app

`)

	start_compliance_example = templates.Examples(`
		# Start compliance
		jx start compliance <org/repo>
	`)
)

// NewCmdStartCompliance creates the command
func NewCmdStartCompliance(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StartComplianceOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,

				Out: out,
				Err: errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "compliance [flags]",
		Short:   "Starts compliance",
		Long:    start_compliance_long,
		Example: start_compliance_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	// TODO once we support get pipelines for Prow we can add support for a selector
	//cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filters all the available jobs by those that contain the given text")

	return cmd
}

// Run implements this command
func (o *StartComplianceOptions) Run() error {
	kClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	if len(o.Args) == 0 {
		return fmt.Errorf("No repsitory specified.\n\n%s", start_compliance_example)
	}
	for _, a := range o.Args {
		err := prow.AddCompliance(kClient, []string{a}, ns)
		if err != nil {
			return err
		}
	}
	log.Infof("Compliance enabled for %s\n", util.ColorInfo(o.Args))
	return nil
}
