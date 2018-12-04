package cmd

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	editStorageLong = templates.LongDesc(`
		Configures the storage location for a set of pipeline output data for your team

		Per team you can specify a Git repository URL to store artifacts inside per classification or you can use a HTTP URL.

		If you don't specify any specific storage it will default to the git repository for a project.'
`)

	editStorageExample = templates.Examples(`
		# To switch your team to helm3 use:
		jx edit storage helm3

		# To switch back to 2.x use:
		jx edit storage helm

	`)
)

// EditStorageOptions the options for the create spring command
type EditStorageOptions struct {
	CreateOptions

	Classifier string
	GitURL     string
	HttpURL    string
}

// NewCmdEditStorage creates a command object for the "create" command
func NewCmdEditStorage(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &EditStorageOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "storage",
		Short:   "Configures the storage location for a set of pipeline output data for your team",
		Aliases: []string{"store"},
		Long:    editStorageLong,
		Example: editStorageExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Classifier, "classifier", "c", "", "A name which classifies this type of file. Example values: "+kube.ClassificationValues)
	cmd.Flags().StringVarP(&options.HttpURL, "http-url", "", "", "Specify the HTTP endpoint to send each file to")
	cmd.Flags().StringVarP(&options.GitURL, "git-url", "", "", "Specify the Git URL to populate in a gh-pages branch")

	return cmd
}

// Run implements the command
func (o *EditStorageOptions) Run() error {
	var err error
	if o.Classifier == "" && ! o.BatchMode {
		o.Classifier, err = util.PickName(kube.Classifications, "Pick the content classification name", "The name is used as a key to store content in different locations", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	if o.Classifier == "" {
		return util.MissingOption("classifier")
	}

	if !o.BatchMode && (o.HttpURL == "" && o.GitURL == "") {
		o.GitURL, err = util.PickValue("Git repository URL to store content:", o.GitURL, false, "The Git URL will be used to clone and push the storage to", o.In, o.Out, o.Err)
		if err != nil {
		  return err
		}
		o.HttpURL, err = util.PickValue("HTTP URL to POST content to:", o.HttpURL, false, "The Git URL will be used to clone and push the storage to", o.In, o.Out, o.Err)
		if err != nil {
		  return err
		}
	}

	callback := func(env *v1.Environment) error {
		location := env.Spec.TeamSettings.StorageLocation(o.Classifier)
		location.GitURL = o.GitURL
		location.HttpUrl = o.HttpURL
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
