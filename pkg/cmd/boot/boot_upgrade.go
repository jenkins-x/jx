package boot

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strings"
)

// BootUpgradeOptions options for the command
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

// NewCmdBootUpgrade creates the command
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

// Run runs this command
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

	err = o.updateBootConfig(o.Dir, reqsVersionStream.URL, reqsVersionStream.Ref, config.DefaultCloudBeesBootRepository)
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

func (o *BootUpgradeOptions) updateBootConfig(dir string, versionStreamURL string, versionStreamRef string, bootConfigURL string) error {
	currentSha, _, _ := o.bootConfigRef(o.Dir, versionStreamURL, versionStreamRef, bootConfigURL)
	upgradeSha, upgradeVersion, _ := o.bootConfigRef(o.Dir, versionStreamURL, "master", bootConfigURL)

	// check if boot config upgrade available
	if upgradeSha == currentSha {
		log.Logger().Infof("No boot config upgrade available")
		os.Exit(1)
	} else {
		log.Logger().Infof("boot config upgrade available!!!!")
	}

	configCloneDir, err := o.cloneBootConfig(bootConfigURL, upgradeVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to clone boot config repo %s", bootConfigURL)
	}

	err = o.cherryPickCommits(o.Dir, configCloneDir, currentSha, upgradeSha, "upgrade_branch")
	if err != nil {
		return errors.Wrap(err, "failed to cherry pick upgrade commits")
	}
	return nil
}

func (o *BootUpgradeOptions) bootConfigRef(dir string, versionStreamURL string, versionStreamRef string, configURL string) (string, string, error) {
	resolver, err := o.CreateVersionResolver(versionStreamURL, versionStreamRef)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to create version resolver %s", configURL)
	}
	configVersion, err := resolver.ResolveGitVersion(configURL)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to resolve config url %s", configURL)
	}
	currentCmtSha, err := o.Git().GetCommitPointedToByTag(dir, fmt.Sprintf("v%s", configVersion))
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to get commit pointed to by %s", currentCmtSha)
	}
	return currentCmtSha, configVersion, nil
}

func (o *BootUpgradeOptions) cloneBootConfig(configURL string, configRef string) (string, error) {
	cloneDir, err := ioutil.TempDir("", "")
	err = os.MkdirAll(cloneDir, util.DefaultWritePermissions)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create directory: %s", cloneDir)
	}

	err = o.Git().Clone(configURL, cloneDir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to clone git URL %s to directory %s", configURL, cloneDir)
	}
	return cloneDir, nil
}

func (o *BootUpgradeOptions) cherryPickCommits(dir, cloneDir, fromSha, toSha, branch string) error {
	cmts := make([]gits.GitCommit, 0)
	cmts, err := o.Git().GetCommits(cloneDir, fromSha, toSha)
	if err != nil {
		return errors.Wrapf(err, "failed to get commits from %s", cloneDir)
	}

	for i := len(cmts) - 1; i >= 0; i-- {
		commitSha := cmts[i].SHA
		log.Logger().Infof("commit sha %s", commitSha)

		err := o.Git().CherryPickTheirs(dir, commitSha)
		if err != nil {
			msg := fmt.Sprintf("commit %s is a merge but no -m option was given.", commitSha)
			if !strings.Contains(err.Error(), msg) {
				return errors.Wrapf(err, "cherry-picking %s", commitSha)
			}
		}
	}
	return nil
}
