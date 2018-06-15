package cmd

import (
	"io"

	"fmt"

	os_user "os/user"

	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"os"
	"path/filepath"
)

// CreateClusterOptions the flags for running create cluster
type UpdateClusterGKETerraformOptions struct {
	UpdateClusterOptions

	Flags UpdateClusterGKETerraformFlags
}

type UpdateClusterGKETerraformFlags struct {
	ClusterName string
}

var (
	updateClusterGKETerraformLong = templates.LongDesc(`

		Command re-applies the terraform plan in ~/.jx/clusters/<cluster>/terraform against the specified cluster

`)

	updateClusterGKETerraformExample = templates.Examples(`

		jx update cluster gke terraform

`)
)

// NewCmdGet creates a command object for the generic "init" action, which
// installs the dependencies required to run the jenkins-x platform on a kubernetes cluster.
func NewCmdUpdateClusterGKETerraform(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := createUpdateClusterGKETerraformOptions(f, out, errOut, GKE)

	cmd := &cobra.Command{
		Use:     "terraform",
		Short:   "Updates an existing kubernetes cluster on GKE using Terraform: Runs on Google Cloud",
		Long:    updateClusterGKETerraformLong,
		Example: updateClusterGKETerraformExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Flags.ClusterName, optionClusterName, "n", "", "The name of this cluster")

	return cmd
}

func createUpdateClusterGKETerraformOptions(f cmdutil.Factory, out io.Writer, errOut io.Writer, cloudProvider string) UpdateClusterGKETerraformOptions {
	commonOptions := CommonOptions{
		Factory: f,
		Out:     out,
		Err:     errOut,
	}
	options := UpdateClusterGKETerraformOptions{
		UpdateClusterOptions: UpdateClusterOptions{
			UpdateOptions: UpdateOptions{
				CommonOptions: commonOptions,
			},
			Provider: cloudProvider,
		},
	}
	return options
}

func (o *UpdateClusterGKETerraformOptions) Run() error {
	err := o.installRequirements(GKE, "terraform")
	if err != nil {
		return err
	}

	err = o.updateClusterGKETerraform()
	if err != nil {
		log.Errorf("error creating cluster %v", err)
		return err
	}

	return nil
}

func (o *UpdateClusterGKETerraformOptions) updateClusterGKETerraform() error {
	if !o.BatchMode {
		confirm := false
		prompt := &survey.Confirm{
			Message: "Updating a GKE cluster with terraform is an experimental feature in jx.  Would you like to continue?",
		}
		survey.AskOne(prompt, &confirm, nil)

		if !confirm {
			// exit at this point
			return nil
		}
	}

	var err error

	if o.Flags.ClusterName == "" {
		log.Info("No cluster name provided\n")
		return nil
	}

	serviceAccount := fmt.Sprintf("jx-%s", o.Flags.ClusterName)

	user, err := os_user.Current()
	if err != nil {
		return err
	}

	jxHome := filepath.Join(user.HomeDir, ".jx")
	clustersHome := filepath.Join(jxHome, "clusters")
	clusterHome := filepath.Join(clustersHome, o.Flags.ClusterName)
	os.MkdirAll(clusterHome, os.ModePerm)

	keyPath := filepath.Join(clusterHome, fmt.Sprintf("%s.key.json", serviceAccount))

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Infof("Unable to find service account key %s\n", keyPath)
		return nil
	}

	terraformDir := filepath.Join(clusterHome, "terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		log.Infof("Unable to find terraform plan dir %s\n", terraformDir)
		return nil
	}

	// create .tfvars file in .jx folder
	terraformVars := filepath.Join(terraformDir, "terraform.tfvars")

	args := []string{"init", terraformDir}
	err = o.runCommand("terraform", args...)
	if err != nil {
		return err
	}

	terraformState := filepath.Join(terraformDir, "terraform.tfstate")

	args = []string{"plan",
		fmt.Sprintf("-state=%s", terraformState),
		fmt.Sprintf("-var-file=%s", terraformVars),
		terraformDir}

	err = o.runCommandVerbose("terraform", args...)
	if err != nil {
		return err
	}

	if !o.BatchMode {
		confirm := false
		prompt := &survey.Confirm{
			Message: "Would you like to apply this plan",
		}
		survey.AskOne(prompt, &confirm, nil)

		if !confirm {
			// exit at this point
			return nil
		}
	}

	log.Info("Applying plan...\n")

	args = []string{"apply",
		"-auto-approve",
		fmt.Sprintf("-state=%s", terraformState),
		fmt.Sprintf("-var-file=%s", terraformVars),
		terraformDir}

	err = o.runCommandVerbose("terraform", args...)
	if err != nil {
		return err
	}

	return nil
}
