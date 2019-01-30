package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// DeletePreviewOptions are the flags for delete commands
type DeletePreviewOptions struct {
	PreviewOptions
}

// NewCmdDeletePreview creates a command object
func NewCmdDeletePreview(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeletePreviewOptions{
		PreviewOptions: PreviewOptions{
			PromoteOptions: PromoteOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Deletes a preview environment",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addPreviewOptions(cmd)
	options.addCommonFlags(cmd)
	return cmd
}

// Run implements this command
func (o *DeletePreviewOptions) Run() error {
	kubeClient, currentNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}

	err = o.defaultValues(ns, o.BatchMode)
	if err != nil {
		if !o.BatchMode {
			jxClient, _, err := o.JXClient()
			if err != nil {
				return err
			}

			// lets let the user pick from a list of preview environments to delete
			names, err := kube.GetFilteredEnvironmentNames(jxClient, ns, kube.IsPreviewEnvironment)
			if err != nil {
				return err
			}
			if len(names) == 0 {
				log.Infof("No preview environments available to delete\n")
				return nil
			}
			selected := []string{}
			for {
				selected, err = util.PickNames(names, "Pick preview environments to delete: ", "", o.In, o.Out, o.Err)
				if err != nil {
					return err
				}
				if len(selected) > 0 {
					break
				}
				log.Warn("\nYou did not select any preview environments to delete\n\n")
				log.Infof("Press the %s to select a preview environment to delete\n\n", util.ColorInfo("[space bar]"))

				if !util.Confirm("Do you want to pick a preview environment to delete?", true, "Use the space bar to select previews", o.In, o.Out, o.Err) {
					return nil
				}
			}
			deletePreviews := strings.Join(selected, ", ")
			if !util.Confirm("You are about to delete the Preview environments: "+deletePreviews, false, "The list of Preview Environments to be deleted", o.In, o.Out, o.Err) {
				return nil
			}

			for _, name := range selected {
				err = o.deletePreview(name)
				if err != nil {
					return err
				}
			}
			return nil
		}
		return err
	}

	if o.Name == "" {
		return fmt.Errorf("Could not default the preview environment name")
	}
	return o.deletePreview(o.Name)
}

func (o *DeletePreviewOptions) deletePreview(name string) error {
	jxClient, ns, err := o.JXClient()
	if err != nil {
		return err
	}

	environment, err := kube.GetEnvironment(jxClient, ns, name)
	if err != nil {
		return err
	}
	releaseName := kube.GetPreviewEnvironmentReleaseName(environment)
	if len(releaseName) > 0 {
		log.Infof("Deleting helm release: %s\n", util.ColorInfo(releaseName))
		err = o.Helm().DeleteRelease(ns, releaseName, true)
		if err != nil {
			return err
		}
	}

	log.Infof("Deleting preview environment: %s\n", util.ColorInfo(name))
	deleteOptions := &DeleteEnvOptions{
		CommonOptions:   o.CommonOptions,
		DeleteNamespace: true,
	}
	deleteOptions.Args = []string{name}
	return deleteOptions.Run()
}
