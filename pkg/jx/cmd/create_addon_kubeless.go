package cmd

import (
	"github.com/spf13/cobra"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	defaultKubelessNamespace   = "kubeless"
	defaultKubelessReleaseName = "kubeless"
)

var (
	createAddonKubelessLong = templates.LongDesc(`
		Creates the kubeless addon for serverless on kubernetes
`)

	createAddonKubelessExample = templates.Examples(`
		# Create the kubeless addon in the ` + defaultKubelessNamespace + ` namespace 
		jx create addon kubeless

		# Create the kubeless addon in a custom namespace
		jx create addon kubeless -n mynamespace
	`)
)

// CreateAddonKubelessOptions the options for the create spring command
type CreateAddonKubelessOptions struct {
	CreateAddonOptions

	Chart string
}

// NewCmdCreateAddonKubeless creates a command object for the "create" command
func NewCmdCreateAddonKubeless(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonKubelessOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "kubeless",
		Short:   "Create a kubeless addon for hosting git repositories",
		Aliases: []string{"env"},
		Long:    createAddonKubelessLong,
		Example: createAddonKubelessExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultKubelessNamespace, defaultKubelessReleaseName)

	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The version of the kubeless addon to use")
	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartKubeless, "The name of the chart to use")
	return cmd
}

// Run implements the command
func (o *CreateAddonKubelessOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	values := []string{"rbac.create=true"}
	err := o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values)
	if err != nil {
		return err
	}
	return nil
}
