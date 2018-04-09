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
	defaultAnchoreNamespace   = "anchore"
	defaultAnchoreReleaseName = "anchore"
	defaultAnchoreVersion     = "0.1.4"
	defaultAnchorePassword    = "anchore"
	defaultAnchoreConfigDir   = "/anchore_service_dir"
)

var (
	createAddonAnchoreLong = templates.LongDesc(`
		Creates the anchore addon for serverless on kubernetes
`)

	createAddonAnchoreExample = templates.Examples(`
		# Create the anchore addon 
		jx create addon anchore

		# Create the anchore addon in a custom namespace
		jx create addon anchore -n mynamespace
	`)
)

// CreateAddonAnchoreOptions the options for the create spring command
type CreateAddonAnchoreOptions struct {
	CreateAddonOptions

	Chart     string
	Password  string
	ConfigDir string
}

// NewCmdCreateAddonAnchore creates a command object for the "create" command
func NewCmdCreateAddonAnchore(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonAnchoreOptions{
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
		Use:     "anchore",
		Short:   "Create the Anchore addon for verifying container images",
		Aliases: []string{"env"},
		Long:    createAddonAnchoreLong,
		Example: createAddonAnchoreExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultAnchoreNamespace, defaultAnchoreReleaseName)

	cmd.Flags().StringVarP(&options.Version, "version", "v", defaultAnchoreVersion, "The version of the Anchore chart to use")
	cmd.Flags().StringVarP(&options.Password, "password", "p", defaultAnchorePassword, "The default password to use for Anchore")
	cmd.Flags().StringVarP(&options.ConfigDir, "config-dir", "d", defaultAnchoreConfigDir, "The config directory to use")
	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartAnchore, "The name of the chart to use")
	return cmd
}

// Run implements the command
func (o *CreateAddonAnchoreOptions) Run() error {
	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	values := []string{"globalConfig.users.admin.password=" + o.Password, "globalConfig.configDir=/anchore_service_dir"}
	err := o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values)
	if err != nil {
		return err
	}
	return nil
}
