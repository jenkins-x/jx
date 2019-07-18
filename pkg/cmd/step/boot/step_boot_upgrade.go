package boot

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	version2 "github.com/jenkins-x/jx/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"path/filepath"
	"strings"
)

// StepBootUpgradeOptions contains the command line flags
type StepBootUpgradeOptions struct {
	*opts.CommonOptions
	Dir         string
	Namespace   string
	AutoUpgrade bool
}

var (
	stepBootUpgradeLong = templates.LongDesc(`
		This step checks the version stream for updates to jenkins-x charts, by default the user is prompted to accept each update
`)

	stepBootUpgradeExample = templates.Examples(`
		# Checks the version stream for updates
		jx step boot upgrade

        # Checks the version stream for updates
		jx step boot upgrade --auto-upgrade
`)
)

// NewCmdStepBootUpgrade creates the command
func NewCmdStepBootUpgrade(commonOpts *opts.CommonOptions) *cobra.Command {
	o := StepBootUpgradeOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "This step checks the version stream for updates to jenkins-x charts",
		Long:    stepBootUpgradeLong,
		Example: stepBootUpgradeExample,
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", fmt.Sprintf("the directory to look for the requirements file: %s", config.RequirementsConfigFileName))
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace that Jenkins X will be booted into. If not specified it defaults to $DEPLOY_NAMESPACE")
	cmd.Flags().BoolVarP(&o.AutoUpgrade, "auto-upgrade", "", false, "Auto apply upgrades")
	return cmd
}

// Run runs the command
func (o *StepBootUpgradeOptions) Run() error {
	ns, err := o.GetDeployNamespace(o.Namespace)
	if err != nil {
		return err
	}
	o.SetDevNamespace(ns)
	requirements, _, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return err
	}
	versionsRepoDir, err := o.CloneJXVersionsRepo(requirements.VersionStream.URL, requirements.VersionStream.Ref)
	if err != nil {
		return errors.Wrapf(err, "cloning the jx versions repo")
	}
	path := "env/requirements.yaml"
	platformRequirements, err := helm.LoadRequirementsFile(path)
	if err != nil {
		message := fmt.Sprintf("loading %s", path)
		return errors.Wrapf(err, message)
	}
	versionsUpdates := false
	for depIndex := range platformRequirements.Dependencies {
		dep := platformRequirements.Dependencies[depIndex]
		glob := filepath.Join(versionsRepoDir, string(version2.KindChart), "*", dep.Name+".yml")
		paths, err := filepath.Glob(glob)
		if err != nil {
			return errors.Wrapf(err, "failed to find chart dependency %s in version stream", dep.Name)
		}
		if len(paths) > 1 {
			log.Logger().Warnf("%s is listed multiple times in the version stream", dep.Name)
			continue
		}
		if len(paths) < 1 {
			log.Logger().Warnf("%s is not listed in the version stream", dep.Name)
			continue
		}
		pathParts := strings.Split(paths[0], "/")
		chartName := pathParts[len(pathParts)-2] + "/" + strings.Replace(pathParts[len(pathParts)-1], ".yml", "", -1)
		newVersion, err := version2.LoadStableVersionNumber(versionsRepoDir, version2.KindChart, chartName)
		if err != nil {
			return errors.Wrapf(err, "failed to load version of chart %s in dir %s", chartName, versionsRepoDir)
		}
		if newVersion > dep.Version {
			message := fmt.Sprintf("Would you like to upgrade %s from version %s to %s?", chartName, dep.Version, newVersion)
			helpMessage := fmt.Sprintf("A new version of %s is available, would you like to upgrade?", chartName)
			if o.AutoUpgrade || (!o.BatchMode && util.Confirm(message, false, helpMessage, o.In, o.Out, o.Err)) {
				log.Logger().Infof("%s was upgraded from %s to %s", chartName, dep.Version, newVersion)
				dep.Version = newVersion
				versionsUpdates = true
			}
		}
	}
	if versionsUpdates {
		err = helm.SaveFile(path, *platformRequirements)
		if err != nil {
			return errors.Wrapf(err, "saving %s", path)
		}
	}
	return nil
}
