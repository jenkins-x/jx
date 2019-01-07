package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/kube/services"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
)

// DeleteAddonOptions are the flags for delete commands
type DeleteAddonOptions struct {
	CommonOptions

	Purge bool
}

// NewCmdDeleteAddon creates a command object for the generic "get" action, which
// retrieves one or more resources from a server.
func NewCmdDeleteAddon(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteAddonOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "addon",
		Short: "Deletes one or more addons",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm"},
	}

	cmd.AddCommand(NewCmdDeleteAddonCloudBees(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteAddonGitea(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteAddonSSO(f, in, out, errOut))
	cmd.AddCommand(NewCmdDeleteAddonKnativeBuild(f, in, out, errOut))
	options.addFlags(cmd)
	return cmd
}

func (options *DeleteAddonOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&options.Purge, "purge", "p", true, "Removes the release name from helm so it can be reused again")
}

// Run implements this command
func (o *DeleteAddonOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return o.Cmd.Help()
	}
	charts := kube.AddonCharts

	for _, arg := range args {
		chart := charts[arg]
		if chart == "" {
			return util.InvalidArg(arg, util.SortedMapKeys(charts))
		}
		err := o.deleteChart(arg, o.Purge)
		if err != nil {
			return fmt.Errorf("Failed to delete chart %s: %s", chart, err)
		}
		err = o.cleanupServiceLink(arg)
		if err != nil {
			return fmt.Errorf("Failed to delete the service link for addon %s", arg)
		}
	}

	return nil
}

func (o *DeleteAddonOptions) cleanupServiceLink(addonName string) error {
	serviceName, ok := kube.AddonServices[addonName]
	if !ok {
		// No cleanup is required if no service link is associated with the Addon
		return nil
	}
	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	svc, err := services.FindService(client, serviceName)
	if err != nil {
		return err
	}

	return client.CoreV1().Services(svc.GetNamespace()).Delete(svc.GetName(), &meta_v1.DeleteOptions{})
}
