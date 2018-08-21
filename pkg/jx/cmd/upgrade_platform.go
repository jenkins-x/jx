package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
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
	Set         string

	InstallFlags InstallFlags
}

// NewCmdUpgradePlatform defines the command
func NewCmdUpgradePlatform(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
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
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The Namespace to promote to")
	cmd.Flags().StringVarP(&options.ReleaseName, "name", "n", "jenkins-x", "The release name")
	cmd.Flags().StringVarP(&options.Chart, "chart", "c", "jenkins-x/jenkins-x-platform", "The Chart to upgrade")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The specific platform version to upgrade to")
	cmd.Flags().StringVarP(&options.Set, "set", "s", "", "The helm parameters to pass in while upgrading")

	options.addCommonFlags(cmd)
	options.InstallFlags.addCloudEnvOptions(cmd)

	return cmd
}

// Run implements the command
func (o *UpgradePlatformOptions) Run() error {
	ns := o.Namespace
	version := o.Version
	err := o.Helm().UpdateRepo()
	if err != nil {
		return err
	}
	if version == "" {
		io := &InstallOptions{}
		io.CommonOptions = o.CommonOptions
		io.Flags = o.InstallFlags
		wrkDir, err := io.cloneJXCloudEnvironmentsRepo()
		if err != nil {
			return err
		}
		version, err = LoadVersionFromCloudEnvironmentsDir(wrkDir)
		if err != nil {
			return err
		}
	}
	if version != "" {
		log.Infof("Upgrading to version %s\n", util.ColorInfo(version))
	}

	dir, err := util.ConfigDir()
	if err != nil {
		return errors.Wrap(err, "failed to create a temporary config dir for git credentials")
	}
	valueFiles := []string{}
	curDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "failed to get the current working directory")
	}
	myValuesFile := filepath.Join(curDir, "myvalues.yaml")
	exists, err := util.FileExists(myValuesFile)
	if err != nil {
		return errors.Wrap(err, "failed to check if the myvaules.yaml file exists in the current directory")
	}
	if exists {
		valueFiles = append(valueFiles, myValuesFile)
		log.Infof("Using local value overrides file %s\n", util.ColorInfo(myValuesFile))
	} else {
		myValuesFile = filepath.Join(dir, "myvalues.yaml")
		exists, err = util.FileExists(myValuesFile)
		if err != nil {
			return errors.Wrap(err, "failed to check if the myvaules.yaml file exists in the .jx directory")
		}
		if exists {
			valueFiles = append(valueFiles, myValuesFile)
			log.Infof("Using local value overrides file %s\n", util.ColorInfo(myValuesFile))
		}
	}

	values := []string{}
	if o.Set != "" {
		values = append(values, o.Set)
	}
	return o.Helm().UpgradeChart(o.Chart, o.ReleaseName, ns, nil, false, nil, false, false, values, valueFiles)
}
