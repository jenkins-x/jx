package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type StepPostRunOptions struct {
	StepOptions

	DisableImport bool
	OutDir        string
}

var ()

var (
	StepPostRunLong = templates.LongDesc(`
		This pipeline step executes any post build actions added during Pipeline execution
`)

	StepPostRunExample = templates.Examples(`
		jx step post run
`)
)

// NewCmdStep Steps a command object for the "step" command
func NewCmdStepPostRun(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepPostRunOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Runs any post build actions",
		Long:    StepPostRunLong,
		Example: StepPostRunExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Verbose, "verbose", "", false, "Enables verbose logging")
	return cmd
}

// Run implements this command
func (o *StepPostRunOptions) Run() (err error) {
	// TODO Support for conditions other than Always
	client, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "cannot create the JX client")
	}

	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterPipelineActivityCRD(apisClient)
	if err != nil {
		return err
	}

	gitInfo, err := o.FindGitInfo("")
	appName := ""
	if gitInfo != nil {
		appName = gitInfo.Name
	}
	pipeline := ""
	build := o.getBuildNumber()
	pipeline, build = o.getPipelineName(gitInfo, pipeline, build, appName)
	if pipeline != "" && build != "" {
		name := kube.ToValidName(pipeline + "-" + build)
		key := &kube.PromoteStepActivityKey{
			PipelineActivityKey: kube.PipelineActivityKey{
				Name:     name,
				Pipeline: pipeline,
				Build:    build,
			},
		}
		a, _, err := key.GetOrCreate(client, ns)
		if err != nil {
			return err
		}
		for _, pe := range a.Spec.PostExtensions {
			log.Infof("Running Extension %s\n", util.ColorInfo(fmt.Sprintf("%s.%s", pe.Namespace, pe.Name)))
			err = pe.Execute(o.Verbose)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
