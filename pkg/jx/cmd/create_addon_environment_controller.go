package cmd

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
)

var (
	createAddonEnvironmentControllerLong = templates.LongDesc(`
		Create an Environment Controller to handle webhooks and promote changes from GitOps 
`)

	createAddonEnvironmentControllerExample = templates.Examples(`
		# Create the Gloo addon 
		jx create addon gloo
	`)
)

// CreateAddonEnvironmentControllerOptions the options for the create spring command
type CreateAddonEnvironmentControllerOptions struct {
	CreateAddonOptions

	Namespace   string
	Version     string
	ReleaseName string
	SetValues   string
	Timeout     int
}

// NewCmdCreateAddonEnvironmentController creates a command object for the "create" command
func NewCmdCreateAddonEnvironmentController(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateAddonEnvironmentControllerOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "environment controller",
		Short:   "Create an Environment Controller to handle webhooks and promote changes from GitOps",
		Aliases: []string{"envctl"},
		Long:    createAddonEnvironmentControllerLong,
		Example: createAddonEnvironmentControllerExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to install the controller")
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", "jx", "The chart release name")
	cmd.Flags().StringVarP(&options.SetValues, "set", "s", "", "The chart set values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	cmd.Flags().StringVarP(&options.Version, "version", "", "", "The version of the chart to use - otherwise the latest version is used")
	cmd.Flags().IntVarP(&options.Timeout, "timeout", "t", 600000, "The timeout value for how long to wait for the install to succeed")
	return cmd
}

// Run implements the command
func (o *CreateAddonEnvironmentControllerOptions) Run() error {
	_, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	if o.Namespace == "" {
		o.Namespace = ns
	}

	helmer := o.NewHelm(false, "", true, true)
	setValues := strings.Split(o.SetValues, ",")
	log.Infof("installing the Environment Controller...\n")
	err = helmer.InstallChart("environment-controller", o.ReleaseName, ns, o.Version, o.Timeout, setValues, nil, kube.DefaultChartMuseumURL, "", "")
	if err != nil {
		return err
	}
	log.Infof("installed the Environment Controller!\n")
	return nil
}
