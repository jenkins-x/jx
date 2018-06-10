package cmd

import (
	"fmt"
	"io"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// CreateAddonOptions the options for the create spring command
type CreateAddonOptions struct {
	CreateOptions

	Namespace   string
	Version     string
	ReleaseName string
	HelmUpdate  bool
}

// NewCmdCreateAddon creates a command object for the "create" command
func NewCmdCreateAddon(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "addon",
		Short:   "Creates an addon",
		Aliases: []string{"scm"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateAddonAmbassador(f, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonAnchore(f, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonCloudBees(f, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonGitea(f, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonIstio(f, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonKubeless(f, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonPipelineEvents(f, out, errOut))

	options.addFlags(cmd, "", "")
	return cmd
}

func (options *CreateAddonOptions) addFlags(cmd *cobra.Command, defaultNamespace string, defaultOptionRelease string) {
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", defaultNamespace, "The Namespace to install into")
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", defaultOptionRelease, "The chart release name")
	cmd.Flags().BoolVarP(&options.HelmUpdate, "helm-update", "", true, "Should we run helm update first to ensure we use the latest version")
}

// Run implements this command
func (o *CreateAddonOptions) Run() error {
	args := o.Args
	if len(args) == 0 {
		return o.Cmd.Help()
	}

	for _, arg := range args {
		err := o.CreateAddon(arg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *CreateAddonOptions) CreateAddon(addon string) error {
	charts := kube.AddonCharts
	chart := charts[addon]
	if chart == "" {
		return util.InvalidArg(addon, util.SortedMapKeys(charts))
	}
	err := o.installChart(addon, chart, o.Version, o.Namespace, o.HelmUpdate, nil)
	if err != nil {
		return fmt.Errorf("Failed to install chart %s: %s", chart, err)
	}
	return nil
}
