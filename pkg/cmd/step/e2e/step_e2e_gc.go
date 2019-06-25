package e2e

import (
	"encoding/json"
	"errors"
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/cmd/deletecmd"
	"github.com/jenkins-x/jx/pkg/cmd/get"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"os"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/spf13/cobra"
)

// StepE2EGCOptions contains the command line flags
type StepE2EGCOptions struct {
	opts.StepOptions
	ProjectID string
	Region    string
	Duration  int
}

// GkeCluster struct to represent a cluster on gcloud
type GkeCluster struct {
	Name           string            `json:"name,omitempty"`
	ResourceLabels map[string]string `json:"resourceLabels,omitempty"`
	Status         string            `json:"status,omitempty"`
}

var (
	stepE2EGCLong = templates.LongDesc(`
		This pipeline step removes stale E2E test clusters
`)

	stepE2EGCExample = templates.Examples(`
		# delete stale E2E test clusters
		jx step e2e gc

`)
)

// NewCmdStepE2EGC creates the CLI command
func NewCmdStepE2EGC(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepE2EGCOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "gc",
		Short:   "Removes unused e2e clusters",
		Aliases: []string{},
		Long:    stepE2EGCLong,
		Example: stepE2EGCExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Region, "region", "", "europe-west1-c", "GKE region to use. Default: europe-west1-c")
	cmd.Flags().StringVarP(&options.ProjectID, "project-id", "p", "", "Google Project ID to delete cluster from")
	cmd.Flags().IntVarP(&options.Duration, "duration", "d", 2, "How many hours old a cluster should be before it is deleted if it does not have a --delete tag")

	return cmd
}

// Run runs the command
func (o *StepE2EGCOptions) Run() error {
	err := o.InstallRequirements(cloud.GKE)
	if err != nil {
		return err
	}
	gkeSa := os.Getenv("GKE_SA_KEY_FILE")
	if gkeSa == "" {
		return errors.New("please set the GKE service account key via the environment variable GKE_SA_KEY_FILE")
	}

	err = gke.Login(gkeSa, true)
	if err != nil {
		return err
	}

	args := []string{"container", "clusters", "list", "--region=" + o.Region, "--format=json", "--quiet"}
	if o.ProjectID != "" {
		args = append(args, "--project="+o.ProjectID)
	}
	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return err
	}

	clusters := make([]GkeCluster, 0)
	err = json.Unmarshal([]byte(output), &clusters)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		if cluster.Status == "RUNNING" {
			deleteCluster := false
			// Marked for deletion
			if deleteLabel, ok := cluster.ResourceLabels["delete-me"]; ok {
				if deleteLabel == "true" {
					deleteCluster = true
				}
			}
			// TODO delete clusters for the same test run if one already exists
			if branchLabel, ok := cluster.ResourceLabels["branch"]; ok {
				if strings.Contains(branchLabel, "pr-") {
					log.Logger().Debugf("Found cluster for branch %s", branchLabel)
				}
			}
			// Older than 2 hours and not marked to be kept
			if createdTime, ok := cluster.ResourceLabels["create-time"]; ok {
				createdDate, err := time.Parse("Mon-Jan-2-2006-15-04-05", createdTime)
				if err != nil {
					log.Logger().Errorf("Error parsing date for cluster %s", createdTime)
					log.Logger().Error(err)
					continue
				}
				ttlExceededDate := createdDate.Add(time.Duration(o.Duration) * time.Hour)
				now := time.Now()
				if now.After(ttlExceededDate) {
					if _, ok := cluster.ResourceLabels["keep-me"]; !ok {
						deleteCluster = true
					}
				}
			}

			if deleteCluster {
				o.deleteGkeCluster(&cluster)
			}
		}
	}
	return nil
}

func (o *StepE2EGCOptions) deleteGkeCluster(cluster *GkeCluster) {
	deleteOptions := &deletecmd.DeleteGkeOptions{
		GetOptions: get.GetOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	deleteOptions.Args = []string{cluster.Name}
	deleteOptions.ProjectID = o.ProjectID
	deleteOptions.Region = o.Region
	err := deleteOptions.Run()
	if err != nil {
		log.Logger().Error(err)
	} else {
		log.Logger().Infof("Deleted cluster %s", cluster.Name)
	}
}
