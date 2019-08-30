package boot

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// BootOptions options for the command
type BootUpgradeOptions struct {
	*opts.CommonOptions
	Dir string
}

var (
	bootUpgradeLong = templates.LongDesc(`
		This command creates a pr for upgrading a jx boot gitOps cluster, incorporating changes to boot
        config and version stream ref
`)

	bootUpgradeExample = templates.Examples(`
		# create pr for upgrading a jx boot gitOps cluster
		jx boot upgrade
`)
)

// NewCmdBoot creates the command
func NewCmdBootUpgrade(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &BootUpgradeOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "boot upgrade",
		Aliases: []string{"bu"},
		Short:   "Upgrades jx boot config",
		Long:    bootUpgradeLong,
		Example: bootUpgradeExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the Jenkins X Pipeline and requirements")
	return cmd
}

func (o *BootUpgradeOptions) Run() error {
	reqsVersionStream, err := o.requirementsVersionStream(o.Dir)
	if err != nil {
		return err
	}

	upgradeVersionSha, err := o.upgradeAvailable(reqsVersionStream.URL, reqsVersionStream.Ref, "master")
	if err != nil {
		return err
	}
	if upgradeVersionSha == "" {
		return nil
	}

	err = o.checkoutNewBranch(o.Dir, "upgrade_branch")
	if err != nil {
		return err
	}

	err = o.updateVersionStreamRef(o.Dir, upgradeVersionSha)
	if err != nil {
		return err
	}

	return nil
}

func (o *BootUpgradeOptions) requirementsVersionStream(dir string) (*config.VersionStreamConfig, error) {
	requirements, requirementsFile, err := config.LoadRequirementsConfig(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to load requirements confif %s", requirementsFile)
	}
	exists, err := util.FileExists(requirementsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file %s exists", requirementsFile)
	}
	if !exists {
		return nil, fmt.Errorf("no requirements file %s ensure you are running this command inside a GitOps clone", requirementsFile)
	}
	reqsVersionSteam := requirements.VersionStream
	return &reqsVersionSteam, nil
}

func (o *BootUpgradeOptions) upgradeAvailable(versionStreamURL string, versionStreamRef string, upgradeRef string) (string, error) {
	versionsDir, err := o.CloneJXVersionsRepo(versionStreamURL, upgradeRef)
	if err != nil {
		return "", errors.Wrapf(err, "failed to clone versions repo %s", versionStreamURL)
	}
	upgradeVersionSha, err := o.Git().GetCommitPointedToByTag(versionsDir, upgradeRef)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get commit pointed to by %s", upgradeRef)
	}

	if versionStreamRef == upgradeVersionSha {
		log.Logger().Infof("No versions stream upgrade available")
		return "", nil
	}
	log.Logger().Infof("versions stream upgrade available!!!!")
	return upgradeVersionSha, nil
}

func (o *BootUpgradeOptions) checkoutNewBranch(dir string, branch string) error {
	err := o.Git().CreateBranch(dir, branch)
	if err != nil {
		return errors.Wrapf(err, "failed to create branch %s", branch)
	}
	err = o.Git().Checkout(dir, branch)
	if err != nil {
		return errors.Wrapf(err, "failed to checkout branch %s", branch)
	}
	return nil
}

func (o *BootUpgradeOptions) updateVersionStreamRef(dir string, upgradeRef string) error {
	requirements, requirementsFile, err := config.LoadRequirementsConfig(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements file %s", requirementsFile)
	}

	requirements.VersionStream.Ref = upgradeRef
	err = requirements.SaveConfig(requirementsFile)
	if err != nil {
		return errors.Wrapf(err, "failed to write version stream to %s", requirementsFile)
	}
	err = o.Git().AddCommitFile(o.Dir, "feat: upgrade version stream", requirementsFile)
	if err != nil {
		return errors.Wrapf(err, "failed to commit requirements file %s", requirementsFile)
	}
	return nil
}

