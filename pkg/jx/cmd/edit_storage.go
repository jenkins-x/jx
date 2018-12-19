package cmd

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
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

		# Configure the git URL of where to store logs
		jx edit storage -c logs --git-url https://github.com/myorg/mylogs.git'

		# Configure the git URL of where all storage goes to by default unless a specific classifier has a config
		jx edit storage -c default --git-url https://github.com/myorg/mylogs.git'

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
func NewCmdEditStorage(commonOpts *CommonOptions) *cobra.Command {
	options := &EditStorageOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
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

	cmd.Flags().StringVarP(&options.Classifier, "classifier", "c", "", "A name which classifies this type of file. Example values: "+kube.ClassificationValues)
	cmd.Flags().StringVarP(&options.HttpURL, "http-url", "", "", "Specify the HTTP endpoint to send each file to")
	cmd.Flags().StringVarP(&options.GitURL, "git-url", "", "", "Specify the Git URL to populate in a gh-pages branch")

	return cmd
}

// Run implements the command
func (o *EditStorageOptions) Run() error {
	var err error
	if o.Classifier == "" && !o.BatchMode {
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
		location.HttpURL = o.HttpURL
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
