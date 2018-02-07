package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"fmt"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
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
func NewCmdDeleteAddon(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteAddonOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "addon",
		Short: "Deletes one or many addons",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
		SuggestFor: []string{"remove", "rm"},
	}

	cmd.AddCommand(NewCmdDeleteAddonGitea(f, out, errOut))
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
	}
	return nil
}
