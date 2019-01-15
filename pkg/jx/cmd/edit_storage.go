package cmd

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
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

		If you don't specify any specific storage for a classifier it will try the classifier 'default'.If there is still no configuration then it will default to the git repository for a project.'
`)

	editStorageExample = templates.Examples(`
		# Be prompted what classification to edit
		jx edit storage

		# Configure the git/http URLs of where to store logs
		jx edit storage -c logs

		# Configure the git URL of where to store logs (defaults to gh-pages branch)
		jx edit storage -c logs --git-url https://github.com/myorg/mylogs.git'

		# Configure the git URL and branch of where to store logs
		jx edit storage -c logs --git-url https://github.com/myorg/mylogs.git' --git-branch cheese

		# Configure the git URL of where all storage goes to by default unless a specific classifier has a config
		jx edit storage -c default --git-url https://github.com/myorg/mylogs.git'

	`)
)

// EditStorageOptions the options for the create spring command
type EditStorageOptions struct {
	CreateOptions

	StorageLocation jenkinsv1.StorageLocation
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

	cmd.Flags().StringVarP(&options.StorageLocation.Classifier, "classifier", "c", "", "A name which classifies this type of file. Example values: "+kube.ClassificationValues)
	cmd.Flags().StringVarP(&options.StorageLocation.HttpURL, "http-url", "", "", "Specify the HTTP endpoint to send each file to")
	cmd.Flags().StringVarP(&options.StorageLocation.GitURL, "git-url", "", "", "Specify the Git URL to populate in a gh-pages branch")
	cmd.Flags().StringVarP(&options.StorageLocation.GitBranch, "git-branch", "", "gh-pages", "The branch to use to store files in the git branch")

	return cmd
}

// Run implements the command
func (o *EditStorageOptions) Run() error {
	settings, err := o.TeamSettings()
	if err != nil {
	  return err
	}

	classifier := o.StorageLocation.Classifier
	if classifier == "" && ! o.BatchMode {
		o.StorageLocation.Classifier, err = util.PickName(kube.Classifications, "Pick the content classification name", "The name is used as a key to store content in different locations", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	if classifier == "" {
		return util.MissingOption("classifier")
	}

	currentLocation := settings.StorageLocationOrDefault(classifier)

	if o.StorageLocation.HttpURL == "" && o.StorageLocation.GitURL == "" {
		if o.BatchMode {
			if currentLocation.GitURL == "" {
				return util.MissingOption("git-url")
			}
			o.StorageLocation.GitURL = currentLocation.GitURL
		} else {
			o.StorageLocation.GitURL, err = util.PickValue("Git repository URL to store content:", currentLocation.GitURL, false, "The Git URL will be used to clone and push the storage to", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
		if o.StorageLocation.GitURL == "" {
			o.StorageLocation.HttpURL, err = util.PickValue("HTTP URL to POST content to:", o.StorageLocation.HttpURL, false, "The Git URL will be used to clone and push the storage to", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
	}

	callback := func(env *v1.Environment) error {
		env.Spec.TeamSettings.SetStorageLocation(classifier, o.StorageLocation)
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
