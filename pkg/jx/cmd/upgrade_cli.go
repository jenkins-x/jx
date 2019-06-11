package cmd

import (
	"encoding/json"
	"github.com/jenkins-x/jx/pkg/jx/cmd/create"
	"runtime"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
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
	create.CreateOptions

	Version string
}

// BrewInfo contains some of the `brew info` data.
type brewInfo struct {
	Name     string
	Outdated bool
	Versions struct {
		Stable string
	}
}

// NewCmdUpgradeCLI defines the command
func NewCmdUpgradeCLI(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &UpgradeCLIOptions{
		CreateOptions: create.CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "cli",
		Short:   "Upgrades the jx command line application - if there are is a new version available",
		Aliases: []string{"client", "clients"},
		Long:    upgradeCLILong,
		Example: upgradeCLIExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The specific version to upgrade to (requires --no-brew on macOS)")
	return cmd
}

// Run implements the command
func (o *UpgradeCLIOptions) Run() error {
	candidateInstallVersion, err := o.candidateInstallVersion()
	if err != nil {
		return err
	}

	currentVersion, err := version.GetSemverVersion()
	if err != nil {
		return errors.Wrap(err, "failed to determine version of currently install jx release")
	}

	log.Logger().Debugf("Current version of jx: %s", util.ColorInfo(currentVersion))

	if o.needsUpgrade(currentVersion, *candidateInstallVersion) {
		return o.InstallJx(true, candidateInstallVersion.String())
	}

	return nil
}

func (o *UpgradeCLIOptions) candidateInstallVersion() (*semver.Version, error) {
	var candidateInstallVersion *semver.Version

	if o.Version == "" {
		latestVersion, err := o.latestAvailableJxVersion()
		if err != nil {
			return nil, errors.Wrap(err, "failed to determine version of latest jx release")
		}
		candidateInstallVersion = latestVersion
	} else {
		if runtime.GOOS == "darwin" && !o.NoBrew {
			return nil, errors.New("requesting an explicit version implies the use of --no-brew")
		}
		requestedVersion, err := semver.New(o.Version)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid version requested: %s", o.Version)
		}
		candidateInstallVersion = requestedVersion
	}
	return candidateInstallVersion, nil
}

func (o *UpgradeCLIOptions) needsUpgrade(currentVersion semver.Version, latestVersion semver.Version) bool {
	if latestVersion.EQ(currentVersion) {
		log.Logger().Infof("You are already on the latest version of jx %s", util.ColorInfo(currentVersion.String()))
		return false
	}
	return true
}

func (o *UpgradeCLIOptions) latestAvailableJxVersion() (*semver.Version, error) {
	var newVersion *semver.Version
	if runtime.GOOS == "darwin" && !o.NoBrew {
		brewInfo, err := o.GetCommandOutput("", "brew", "info", "--json", "jx")
		if err != nil {
			return nil, err
		}

		v, err := o.latestJxBrewVersion(brewInfo)
		if err != nil {
			return nil, err
		}

		newVersion, err = semver.New(v)
		if err != nil {
			return nil, err
		}
		log.Logger().Debugf("Found the latest Homebrew released version of jx: %s", util.ColorInfo(newVersion))
	} else {
		v, err := o.GetLatestJXVersion()
		if err != nil {
			return nil, err
		}
		newVersion = &v
		log.Logger().Debugf("Found the latest GitHub released version of jx: %s", util.ColorInfo(newVersion))
	}
	return newVersion, nil
}

func (o *UpgradeCLIOptions) latestJxBrewVersion(jsonInfo string) (string, error) {
	var brewInfo []brewInfo
	err := json.Unmarshal([]byte(jsonInfo), &brewInfo)
	if err != nil {
		return "", err
	}
	return brewInfo[0].Versions.Stable, nil
}
