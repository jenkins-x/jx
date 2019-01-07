package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/sirupsen/logrus"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	deleteAddonKnativeBuildLong = templates.LongDesc(`
		Deletes the KnativeBuild addon
`)

	deleteAddonKnativeBuildExample = templates.Examples(`
		# Deletes the KnativeBuild addon
		jx delete addon KnativeBuild
	`)
)

// DeleteAddonGiteaOptions the options for the create spring command
type DeleteKnativeBuildOptions struct {
	DeleteAddonOptions

	ReleaseName string
}

// NewCmdDeleteAddonKnativeBuild defines the command
func NewCmdDeleteAddonKnativeBuild(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteKnativeBuildOptions{
		DeleteAddonOptions: DeleteAddonOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "knative-build",
		Short:   "Deletes the KnativeBuild app for Kubernetes addon",
		Aliases: []string{"cloudbee", "cb", "core"},
		Long:    deleteAddonKnativeBuildLong,
		Example: deleteAddonKnativeBuildExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", kube.DefaultKnativeBuildReleaseName, "The chart release name")
	options.addFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteKnativeBuildOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}

	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	knativeCRDs := []string{"clusterbuildtemplates.build.knative.dev", "images.caching.internal.knative.dev", "buildtemplates.build.knative.dev", "builds.build.knative.dev"}

	for _, crd := range knativeCRDs {
		err = apisClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd, &metav1.DeleteOptions{})
		if err != nil {
			logrus.Warnf("cannot delete CRD %s: %v", crd, err)
			confirm := &survey.Confirm{
				Message: "There are warnings, do you wish to continue?",
				Default: false,
			}
			flag := true
			err = survey.AskOne(confirm, &flag, nil)
			if err != nil || flag == false {
				return nil
			}
		}
	}
	err = o.deleteChart(o.ReleaseName, o.Purge)
	if err != nil {
		return err
	}

	return nil
}
