package cmd

import (
	"io"
	"runtime"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/spf13/cobra"
)

var (
	upgradeCLILong = templates.LongDesc(`
		Upgrades the Jenkins X command line tools if there is a newer release
`)

	upgradeCLIExample = templates.Examples(`
		# Upgrades the Jenkins X CLI tools 
		jx upgrade cli
	`)
)

// UpgradeCLIOptions the options for the create spring command
type UpgradeCLIOptions struct {
	CreateOptions

	Version string
}

// NewCmdUpgradeCLI defines the command
func NewCmdUpgradeCLI(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &UpgradeCLIOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "cli",
		Short:   "Upgrades the command line applications - if there are new versions available",
		Aliases: []string{"client", "clients"},
		Long:    upgradeCLILong,
		Example: upgradeCLIExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The specific version to upgrade to")
	return cmd
}

// Run implements the command
func (o *UpgradeCLIOptions) Run() error {
	newVersion, err := o.getLatestJXVersion()
	if err != nil {
		return err
	}

	currentVersion, err := version.GetSemverVersion()
	if err != nil {
		return err
	}

	if newVersion.EQ(currentVersion) {
		log.Infof("You are already on the latest version of jx %s\n", util.ColorInfo(currentVersion.String()))
		return nil
	}
	if newVersion.LE(currentVersion) {
		log.Infof("Your jx version %s is actually newer than the latest available version %s\n", util.ColorInfo(currentVersion.String()), util.ColorInfo(newVersion.String()))
		return nil
	}

	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.runCommand("brew", "upgrade", "jx")
	} else {
		return o.installJx(true, newVersion.String())
	}
}
