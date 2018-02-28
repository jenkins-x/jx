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

	cmd.AddCommand(NewCmdCreateAddonCDX(f, out, errOut))
	cmd.AddCommand(NewCmdCreateAddonGitea(f, out, errOut))

	options.addFlags(cmd)
	return cmd
}

func (options *CreateAddonOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The Namespace to install into")
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", "gitea", "The chart release name")
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

func (o *CreateAddonOptions) CreateAddon(arg string) error {
	charts := kube.AddonCharts
	chart := charts[arg]
	if chart == "" {
		return util.InvalidArg(arg, util.SortedMapKeys(charts))
	}
	err := o.installChart(arg, chart, o.Version, o.Namespace, o.HelmUpdate)
	if err != nil {
		return fmt.Errorf("Failed to install chart %s: %s", chart, err)
	}
	return nil
}
