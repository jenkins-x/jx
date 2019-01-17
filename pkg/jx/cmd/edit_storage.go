package cmd

import (
	"context"
	"fmt"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gocloud.dev/blob"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"net/url"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	editStorageLong = templates.LongDesc(`
		Configures the storage location used by your team to stashing files or storing build logs.

		If you don't specify any specific storage for a classifier it will try the classifier 'default'. If there is still no configuration then it will default to the git repository for a project.'

		See also:

        * 'jx get storage' command: https://jenkins-x.io/commands/jx_get_storage/
        * 'jx step stash' command: https://jenkins-x.io/commands/jx_step_storage/

` + StorageSupportDescription)

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
	Bucket          string
	BucketKind      string
	GKEProjectID    string
	GKEZone         string
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

	cmd.Flags().StringVarP(&options.StorageLocation.Classifier, "classifier", "c", "", "A name which classifies this type of file. Example values: "+kube.ClassificationValues)
	cmd.Flags().StringVarP(&options.StorageLocation.BucketURL, "bucket-url", "", "", "Specify the go-cloud URL of the bucket to use")
	cmd.Flags().StringVarP(&options.StorageLocation.GitURL, "git-url", "", "", "Specify the Git URL to populate in a gh-pages branch")
	cmd.Flags().StringVarP(&options.StorageLocation.GitBranch, "git-branch", "", "gh-pages", "The branch to use to store files in the git branch")
	cmd.Flags().StringVarP(&options.Bucket, "bucket", "", "", "Specify the name of the bucket to use")
	cmd.Flags().StringVarP(&options.BucketKind, "bucket-kind", "", "", "The kind of bucket to use like 'gs, s3, azure' etc")
	cmd.Flags().StringVarP(&options.GKEProjectID, "gke-project-id", "", "", "Google Project ID to use for a new bucket")
	cmd.Flags().StringVarP(&options.GKEZone, "gke-zone", "", "", "The zone (e.g. us-central1-a) where the new bucket will be created")

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
			return errors.Wrapf(err, "failed to pick the classification name")
		}
	}
	if classifier == "" {
		return util.MissingOption("classifier")
	}

	currentLocation := settings.StorageLocationOrDefault(classifier)

	if o.StorageLocation.BucketURL == "" && o.StorageLocation.GitURL == "" {
		if o.Bucket != "" {
			o.StorageLocation.BucketURL, err = buckets.CreateBucketURL(o.Bucket, o.BucketKind, settings)
			if err != nil {
				return errors.Wrapf(err, "failed to create the bucket URL for %s", o.Bucket)
			}

			ctx, _ := context.WithTimeout(context.Background(), time.Second * 20)
			bucket, err := blob.Open(ctx, o.StorageLocation.BucketURL)
			if err != nil {
				return errors.Wrapf(err, "failed to open the bucket for %s", o.StorageLocation.BucketURL)
			}

			// lets check if the bucket exists
			iter := bucket.List(nil)
			obj, err := iter.Next(ctx)
			if err != nil {
				if err == io.EOF {
					log.Infof("bucket %s is empty\n", o.StorageLocation.BucketURL)
				} else {
					log.Infof("The bucket %s does not exist yet so lets create it...\n", util.ColorInfo(o.StorageLocation.BucketURL))
					err = o.createBucket(o.StorageLocation.BucketURL, bucket)
					if err != nil {
						return errors.Wrapf(err, "failed to create the bucket for %s", o.StorageLocation.BucketURL)
					}
				}
			} else {
				log.Infof("Found item in bucket %s for %s\n", o.StorageLocation.BucketURL, obj.Key)
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

// createBucket creates a bucket if it does not already exist
func (o *EditStorageOptions) createBucket(bucketURL string, bucket *blob.Bucket) error {
	u, err := url.Parse(bucketURL)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "gs":
		return o.createGcsBucket(u, bucket)
	default:
		return fmt.Errorf("Cannot create a bucket for provider %s", bucketURL)
	}
}

func (o *EditStorageOptions) createGcsBucket(u *url.URL, bucket *blob.Bucket) error {
	var err error
	if o.GKEProjectID == "" {
		o.GKEProjectID, err = o.getGoogleProjectId()
		if err != nil {
			return err
		}
	}

	err = o.CreateOptions.CommonOptions.runCommandVerbose(
		"gcloud", "config", "set", "project", o.GKEProjectID)
	if err != nil {
		return err
	}

	if o.GKEZone == "" {
		defaultZone := ""
		if cluster, err := gke.ClusterName(o.Kube()); err == nil && cluster != "" {
			if clusterZone, err := gke.ClusterZone(cluster); err == nil {
				defaultZone = clusterZone
			}
		}

		o.GKEZone, err = o.getGoogleZoneWithDefault(o.GKEProjectID, defaultZone)
		if err != nil {
			return err
		}
	}

	bucketName := u.Host
	region := gke.GetRegionFromZone(o.GKEZone, )
	err = gke.CreateBucket(o.GKEProjectID, bucketName, region)
	if err != nil {
		return errors.Wrapf(err, "creating bucket %s in project %s and region %s", bucketName, o.GKEProjectID, region)
	}
	return nil
}
