package cmd

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
)

var (
	editStorageLong = templates.LongDesc(`
		Configures the storage location used by your team to stashing files or storing build logs.

		If you don't specify any specific storage for a classifier it will try the classifier 'default'. If there is still no configuration then it will default to the git repository for a project.'

` + storageSupportDescription + SeeAlsoText("jx step stash", "jx get storage"))

	editStorageExample = templates.Examples(`
		# Be prompted what classification to edit
		jx edit storage

		# Configure the where to store logs prompting the user to ask for more data
		jx edit storage -c logs


		# Configure the git URL of where to store logs (defaults to gh-pages branch)
		jx edit storage -c logs --git-url https://github.com/myorg/mylogs.git'

		# Configure the git URL and branch of where to store logs
		jx edit storage -c logs --git-url https://github.com/myorg/mylogs.git' --git-branch cheese

		# Configure the git URL of where all storage goes to by default unless a specific classifier has a config
		jx edit storage -c default --git-url https://github.com/myorg/mylogs.git'


		# Configure the tests to be stored in cloud storage (using S3 / GCS / Azure Blobs etc)
		jx edit storage -c tests --bucket-url s3://myExistingBucketName

		# Creates a new GCS bucket and configures the logs to be stored in it
		jx edit storage -c logs --bucket myBucketName
	`)
)

// EditStorageOptions the options for the create spring command
type EditStorageOptions struct {
	CreateOptions

	StorageLocation jenkinsv1.StorageLocation
	CreateBucket    CreateBucketValues
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
		Short:   "Configures the storage location for stashing files or storing build logs for your team",
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
	addStorageLocationFlags(cmd, &options.StorageLocation)

	options.CreateBucket.addCreateBucketFlags(cmd)
	return cmd
}


func addStorageLocationFlags(cmd *cobra.Command, location *jenkinsv1.StorageLocation) {
	cmd.Flags().StringVarP(&location.Classifier, "classifier", "c", "", "A name which classifies this type of file. Example values: "+kube.ClassificationValues)
	cmd.Flags().StringVarP(&location.BucketURL, "bucket-url", "", "", "Specify the cloud storage bucket URL to send each file to. e.g. use 's3://nameOfBucket' on AWS, gs://anotherBucket' on GCP or on Azure 'azblob://thatBucket'")
	cmd.Flags().StringVarP(&location.GitURL, "git-url", "", "", "Specify the Git URL to of the repository to use for storage")
	cmd.Flags().StringVarP(&location.GitBranch, "git-branch", "", "gh-pages", "The branch to use to store files in the git repository")
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
			return errors.Wrapf(err, "failed to pick the classification name")
		}
	}
	if classifier == "" {
		return util.MissingOption("classifier")
	}

	currentLocation := settings.StorageLocationOrDefault(classifier)

	if o.StorageLocation.BucketURL == "" && o.StorageLocation.GitURL == "" {
		if !o.CreateBucket.IsEmpty() {
			o.StorageLocation.BucketURL, err = o.createBucket(&o.CreateBucket, settings)
			if err != nil {
			  return err
			}
		}
		if o.StorageLocation.BucketURL == "" {
			o.StorageLocation.BucketURL, err = util.PickValue("Bucket URL:", o.StorageLocation.BucketURL, false, "The go-cloud bucket URL for storage such as 'gs://mybucket/ or s3://bucket2/", o.In, o.Out, o.Err)
			if err != nil {
				return errors.Wrapf(err, "failed to pick the bucket URL")
			}
		}

		if o.StorageLocation.BucketURL == "" {
			if o.BatchMode {
				if currentLocation.GitURL == "" {
					return util.MissingOption("git-url")
				}
				o.StorageLocation.GitURL = currentLocation.GitURL
			} else {
				o.StorageLocation.GitURL, err = util.PickValue("Git repository URL to store content:", currentLocation.GitURL, false, "The Git URL will be used to clone and push the storage to", o.In, o.Out, o.Err)
				if err != nil {
					return errors.Wrapf(err, "failed to pick the git URL")
				}
			}
		}
	}

	callback := func(env *v1.Environment) error {
		env.Spec.TeamSettings.SetStorageLocation(classifier, o.StorageLocation)
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
