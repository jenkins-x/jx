package cmd

import (
	"io"
	"runtime"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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
func NewCmdUpgradeCLI(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpgradeCLIOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commoncmd.CommonOptions{
				Factory: f,
				In:      in,
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
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The specific version to upgrade to")
	options.AddCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *UpgradeCLIOptions) Run() error {
	log.ConfigureLog(o.LogLevel)

	newVersion, err := o.GetLatestJXVersion()
	if err != nil {
		return err
	}
	logger.Debugf("Found the latest version of jx: %s", util.ColorInfo(newVersion))

	currentVersion, err := version.GetSemverVersion()
	if err != nil {
		return err
	}

	if newVersion.EQ(currentVersion) {
		logger.Infof("You are already on the latest version of jx %s", util.ColorInfo(currentVersion.String()))
		return nil
	}
	if newVersion.LE(currentVersion) {
		logger.Infof("Your jx version %s is actually newer than the latest available version %s", util.ColorInfo(currentVersion.String()), util.ColorInfo(newVersion.String()))
		return nil
	}

	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "upgrade", "jx")
	} else {
		return o.InstallJx(true, newVersion.String())
	}
}
