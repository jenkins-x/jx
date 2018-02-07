package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/spf13/cobra"
)

var (
	upgrade_platform_long = templates.LongDesc(`
		Upgrades the Jenkins X platform if there is a newer release
`)

	upgrade_platform_example = templates.Examples(`
		# Upgrades the Jenkins X platform 
		jx upgrade platform
	`)
)

// UpgradePlatformOptions the options for the create spring command
type UpgradePlatformOptions struct {
	CreateOptions

	Version     string
	ReleaseName string
	Chart       string
	Namespace   string
}

// NewCmdUpgradePlatform defines the command
func NewCmdUpgradePlatform(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &UpgradePlatformOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "platform",
		Short:   "Upgrades the Jenkins X platform if there is a new release available",
		Aliases: []string{"token"},
		Long:    upgrade_platform_long,
		Example: upgrade_platform_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The Namespace to promote to")
	cmd.Flags().StringVarP(&options.ReleaseName, "name", "n", "jenkins-x", "The release name")
	cmd.Flags().StringVarP(&options.Chart, "chart", "c", "jenkins-x/jenkins-x-platform", "The Chart to upgrade")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The Version to promote")
	return cmd
}

// Run implements the command
func (o *UpgradePlatformOptions) Run() error {
	err := o.runCommand("helm", "repo", "update")
	if err != nil {
		return err
	}
	args := []string{"upgrade"}
	version := o.Version
	if version != "" {
		args = append(args, "--version", version)
	}
	ns := o.Namespace
	if ns != "" {
		args = append(args, "--namespace", ns)
	}
	args = append(args, o.ReleaseName, o.Chart)
	return o.runCommand("helm", args...)
}
