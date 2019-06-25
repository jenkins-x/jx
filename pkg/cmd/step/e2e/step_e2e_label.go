package e2e

import (
	"encoding/json"
	"errors"
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"strings"
)

// StepE2ELabelOptions contains the command line flags
type StepE2ELabelOptions struct {
	opts.StepOptions
	ProjectID string
	Region    string
	Keep      bool
	Delete    bool
}

var (
	stepE2EApplyLabelLong = templates.LongDesc(`
		Add a label to a cluster used for e2e testing
`)

	stepE2EApplyLabelExample = templates.Examples(`
		# Mark a cluster to not be deleted by the gc
		jx step e2e label --keep clusterName

        # Mark a cluster to be deleted by the gc
		jx step e2e label --delete clusterName

`)
)

// NewCmdStepE2ELabel creates the CLI command
func NewCmdStepE2ELabel(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepE2ELabelOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "label",
		Short:   "Removes unused e2e clusters",
		Aliases: []string{},
		Long:    stepE2EApplyLabelLong,
		Example: stepE2EApplyLabelExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Region, "region", "", "europe-west1-c", "GKE region to use. Default: europe-west1-c")
	cmd.Flags().StringVarP(&options.ProjectID, "project-id", "p", "", "Google Project ID to delete cluster from")
	cmd.Flags().BoolVarP(&options.Keep, "keep", "k", false, "Add a label top mark cluster for non deletion")
	cmd.Flags().BoolVarP(&options.Delete, "delete", "d", false, "Add a label top mark cluster for deletion")

	return cmd
}

// Run runs the command
func (o *StepE2ELabelOptions) Run() error {
	if len(o.Args) == 0 {
		return errors.New("Please specifiy a cluster name")
	}
	if o.Keep == o.Delete {
		return errors.New("Please specify either --keep or --delete")
	}
	err := o.InstallRequirements(cloud.GKE)
	if err != nil {
		return err
	}
	clusterName := o.Args[0]
	cluster, err := o.loadGkeCluster(clusterName)
	if err != nil {
		return err
	}

	if cluster != nil {
		labelMap := cluster.ResourceLabels
		labels := make([]string, 0)
		if o.Keep {
			labelMap["keep-me"] = "true"
		} else {
			delete(labelMap, "keep-me")
		}
		if o.Delete {
			labelMap["delete-me"] = "true"
		} else {
			delete(labelMap, "delete-me")
		}
		for key, value := range labelMap {
			labels = append(labels, key+"="+value)
		}

		args := []string{"container", "clusters", "update", clusterName, "--region=" + o.Region, "--quiet", "--update-labels=" + strings.Join(labels, ",") + ""}
		if o.ProjectID != "" {
			args = append(args, "--project="+o.ProjectID)
		}
		cmd := util.Command{
			Name: "gcloud",
			Args: args,
		}
		_, err = cmd.RunWithoutRetry()
		if err == nil {
			if o.Keep {
				log.Logger().Infof("%s was marked to be kept", cluster.Name)
			} else {
				log.Logger().Infof("%s was marked to be deleted", cluster.Name)
			}
		} else {
			log.Logger().Error(err)
		}
	}
	return nil
}

func (o *StepE2ELabelOptions) loadGkeCluster(clusterName string) (*GkeCluster, error) {
	args := []string{"container", "clusters", "describe", clusterName, "--region=" + o.Region, "--format=json", "--quiet"}
	if o.ProjectID != "" {
		args = append(args, "--project="+o.ProjectID)
	}
	cmd := util.Command{
		Name: "gcloud",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return nil, err
	}

	cluster := &GkeCluster{}
	err = json.Unmarshal([]byte(output), cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}
