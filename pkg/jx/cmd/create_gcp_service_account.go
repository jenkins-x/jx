package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/util"
	"fmt"
	"errors"
	"github.com/jenkins-x/jx/pkg/log"
)

type CreateGcpServiceAccountFlags struct {
	Name              string
	Project string
}

type CreateGcpServiceAccountOptions struct {
	CreateOptions
	Flags                CreateGcpServiceAccountFlags
}

var (
	createGcpServiceAccountExample = templates.Examples(`
		jx create gcp-service-account

		# to specify the options via flags
		jx create gcp-service-account --name my-service-account --project my-gcp-project

`)
)

// NewCmdCreateGcpServiceAccount creates a command object for the "create" command
func NewCmdCreateGcpServiceAccount(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateGcpServiceAccountOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "gcp-service-account",
		Short:   "Creates a GCP service account",
		Example: createGcpServiceAccountExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd)

	return cmd
}

func (options *CreateGcpServiceAccountOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&options.Flags.Name, "name", "n", "", "The name of the service account to create")
	cmd.Flags().StringVarP(&options.Flags.Project, "project", "p", "", "The GCP project to create the service account in")
}

// Run implements this command
func (o *CreateGcpServiceAccountOptions) Run() error {
	if o.Flags.Name == "" {
		prompt := &survey.Input{
			Message: "Name for the service account",
		}

		err := survey.AskOne(prompt, &o.Flags.Name, nil)
		if err != nil {
			return err
		}
	}

	if o.Flags.Project == "" {
		projectId, err := o.getGoogleProjectId()
		if err != nil {
			return err
		}
		o.Flags.Project = projectId
	}

	path, err := gke.GetOrCreateServiceAccount(o.Flags.Name, o.Flags.Project, util.HomeDir())
	if err != nil {
		return err
	}

	log.Infof("Created service account key %s\n", util.ColorInfo(path))

	return nil
}

// asks to chose from existing projects or optionally creates one if none exist
func (o *CreateGcpServiceAccountOptions) getGoogleProjectId() (string, error) {
	existingProjects, err := gke.GetGoogleProjects()
	if err != nil {
		return "", err
	}

	var projectId string
	if len(existingProjects) == 0 {
		confirm := &survey.Confirm{
			Message: fmt.Sprintf("No existing Google Projects exist, create one now?"),
			Default: true,
		}
		flag := true
		err = survey.AskOne(confirm, &flag, nil)
		if err != nil {
			return "", err
		}
		if !flag {
			return "", errors.New("no google project to create cluster in, please manual create one and rerun this wizard")
		}

		if flag {
			return "", errors.New("auto creating projects not yet implemented, please manually create one and rerun the wizard")
		}
	} else if len(existingProjects) == 1 {
		projectId = existingProjects[0]
		log.Infof("Using the only Google Cloud Project %s to create the cluster\n", util.ColorInfo(projectId))
	} else {
		prompts := &survey.Select{
			Message: "Google Cloud Project:",
			Options: existingProjects,
			Help:    "Select a Google Project to create the cluster in",
		}

		err := survey.AskOne(prompts, &projectId, nil)
		if err != nil {
			return "", err
		}
	}

	if projectId == "" {
		return "", errors.New("no Google Cloud Project to create cluster in, please manual create one and rerun this wizard")
	}

	return projectId, nil
}