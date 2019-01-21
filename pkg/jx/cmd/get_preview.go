package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPreviewOptions containers the CLI options
type GetPreviewOptions struct {
	GetEnvOptions

	Current bool
}

var (
	getPreviewLong = templates.LongDesc(`
		Display one or more environments.
`)

	getPreviewExample = templates.Examples(`
		# List all preview environments
		jx get previews

		# View the current preview environment URL
		# inside a CI pipeline
		jx get preview --current
	`)
)

// NewCmdGetPreview creates the new command for: jx get env
func NewCmdGetPreview(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetPreviewOptions{
		GetEnvOptions: GetEnvOptions{
			GetOptions: GetOptions{
				CommonOptions: commoncmd.CommonOptions{
					Factory: f,
					In:      in,

					Out: out,
					Err: errOut,
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "previews",
		Short:   "Display one or more Preview Environments",
		Aliases: []string{"preview"},
		Long:    getPreviewLong,
		Example: getPreviewExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Current, "current", "c", false, "Output the URL of the current Preview application the current pipeline just deployed")

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetPreviewOptions) Run() error {
	if o.Current {
		return o.CurrentPreviewUrl()
	}
	o.PreviewOnly = true
	return o.GetEnvOptions.Run()
}

func (o *GetPreviewOptions) CurrentPreviewUrl() error {
	pipeline := o.GetJobName()
	if pipeline == "" {
		return fmt.Errorf("No $JOB_NAME defined for the current pipeline job to use")
	}
	name := kube.ToValidName(pipeline)

	client, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	envList, err := client.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, env := range envList.Items {
		if env.Spec.Kind == v1.EnvironmentKindTypePreview && env.Name == name {
			log.Info(env.Spec.PreviewGitSpec.ApplicationURL)
			return nil
		}
	}
	return fmt.Errorf("No Preview for name: %s", name)
}
