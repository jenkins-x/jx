package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// StepHelmEnvOptions contains the command line flags
type StepHelmEnvOptions struct {
	StepHelmOptions

	recursive bool
}

var (
	StepHelmEnvLong = templates.LongDesc(`
		Generates the helm environment variables
`)

	StepHelmEnvExample = templates.Examples(`
		# output the helm environment variables that should be set to use helm directly
		jx step helm env

`)
)

func NewCmdStepHelmEnv(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepHelmEnvOptions{
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
		Use:     "env",
		Short:   "Generates the helm environment variables",
		Aliases: []string{""},
		Long:    StepHelmEnvLong,
		Example: StepHelmEnvExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addStepHelmFlags(cmd)

	return cmd
}

func (o *StepHelmEnvOptions) Run() error {
	h := o.Helm()
	if h != nil {
		log.Info("\n")
		log.Info("# helm environment variables\n")
		envVars := h.Env()
		keys := util.SortedMapKeys(envVars)
		for _, key := range keys {
			if strings.HasPrefix(key, "HELM") {
				log.Infof("export %s=\"%s\"\n", key, envVars[key])
			}
		}
		log.Info("\n")
	}
	return nil
}
