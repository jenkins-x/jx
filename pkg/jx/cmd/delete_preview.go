package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/kube"
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
	return cmd
}

// Run implements this command
func (o *DeletePreviewOptions) Run() error {
	f := o.Factory
	kubeClient, currentNs, err := f.CreateClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}

	err = o.defaultValues(ns)
	if err != nil {
		return err
	}

	if o.Name == "" {
		return fmt.Errorf("Could not default the preview environment name!")
	}

	deleteOptions := &DeleteEnvOptions{
		CommonOptions:   o.CommonOptions,
		DeleteNamespace: true,
	}
	deleteOptions.Args = []string{o.Name}
	return deleteOptions.Run()
}
