package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	editHelmBinLong = templates.LongDesc(`
		Configures the helm binary version used by your team

		This lets you switch between helm and helm3
`)

	editHelmBinExample = templates.Examples(`
		# To switch your team to helm3 use:
		jx edit helmbin helm3

		# To switch back to 2.x use:
		jx edit helmbin helm

	`)
)

// EditHelmBinOptions the options for the create spring command
type EditHelmBinOptions struct {
	CreateOptions

	BranchPattern string
}

// NewCmdEditHelmBin creates a command object for the "create" command
func NewCmdEditHelmBin(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &EditHelmBinOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "helmbin",
		Short:   "Configures the helm binary version used by your team",
		Aliases: []string{"helm"},
		Long:    editHelmBinLong,
		Example: editHelmBinExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *EditHelmBinOptions) Run() error {
	if len(o.Args) == 0 {
		return fmt.Errorf("Missing argument for the helm binary")
	}
	arg := o.Args[0]

	if !strings.HasPrefix(arg, "helm") {
		return util.InvalidArgError(arg, fmt.Errorf("Helm binary name should start with 'helm'"))
	}

	callback := func(env *v1.Environment) error {
		env.Spec.TeamSettings.HelmBinary = arg
		o.Printf("Setting the helm binary name to: %s\n", util.ColorInfo(arg))
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
