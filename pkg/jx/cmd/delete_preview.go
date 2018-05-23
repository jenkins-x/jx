package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

// DeletePreviewOptions are the flags for delete commands
type DeletePreviewOptions struct {
	PreviewOptions
}

// NewCmdDeletePreview creates a command object
func NewCmdDeletePreview(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeletePreviewOptions{
		PreviewOptions: PreviewOptions{
			PromoteOptions: PromoteOptions{
				CommonOptions: CommonOptions{
					Factory: f,
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
			cmdutil.CheckErr(err)
		},
	}
	options.addPreviewOptions(cmd)
	options.addCommonFlags(cmd)
	return cmd
}

// Run implements this command
func (o *DeletePreviewOptions) Run() error {
	kubeClient, currentNs, err := o.KubeClient()
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
			selected, err := util.PickNames(names, "Pick preview environments to delete: ")
			if err != nil {
				return err
			}
			deletePreviews := strings.Join(selected, ", ")
			if !util.Confirm("You are about to delete the Preview environments: "+deletePreviews, false, "The list of Preview Enviroments to be deleted") {
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
		return fmt.Errorf("Could not default the preview environment name!")
	}
	return o.deletePreview(o.Name)
}

func (o *DeletePreviewOptions) deletePreview(name string) error {
	o.Printf("Deleting preview environment: %s\n", util.ColorInfo(name))
	deleteOptions := &DeleteEnvOptions{
		CommonOptions:   o.CommonOptions,
		DeleteNamespace: true,
	}
	deleteOptions.Args = []string{name}
	return deleteOptions.Run()
}
