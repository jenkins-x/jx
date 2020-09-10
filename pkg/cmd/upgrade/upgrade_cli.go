package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rhysd/go-github-selfupdate/selfupdate"

	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxclient"
	"github.com/jenkins-x/jx/v2/pkg/kube"

	"github.com/jenkins-x/jx-helpers/pkg/versionstream"

	"github.com/blang/semver"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/packages"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/jenkins-x/jx/v2/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdCLILong = templates.LongDesc(`
		Upgrades your local Jenkins X CLI
`)

	cmdCLIExample = templates.Examples(`
		# upgrades your jx CLI
		jx upgrade cli
	`)
)

const (
	// BinaryDownloadBaseURL the base URL for downloading the binary from - will always have "VERSION/jx-OS-ARCH.EXTENSION" appended to it when used
	BinaryDownloadBaseURL = "https://github.com/jenkins-x/jx-cli/releases/download/v"
)

// UpgradeOptions the options for upgrading a cluster
type CLIOptions struct {
	CommandRunner       cmdrunner.CommandRunner
	GitClient           gitclient.Interface
	JXClient            versioned.Interface
	Version             string
	VersionStreamGitURL string
	FromEnvironment     string
}

// NewCmdUpgrade creates a command object for the command
func NewCmdUpgradeCLI() (*cobra.Command, *CLIOptions) {
	o := &CLIOptions{}

	cmd := &cobra.Command{
		Use:     "cli",
		Short:   "Upgrades your local Jenkins X CLI",
		Long:    cmdCLILong,
		Example: cmdCLIExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "The specific version to upgrade to (requires --brew=false on macOS)")
	cmd.Flags().StringVarP(&o.VersionStreamGitURL, "version-stream-git-url", "", "https://github.com/jenkins-x/jxr-versions.git", "The version stream git URL to lookup the jx cli version to upgrade to")
	cmd.Flags().StringVarP(&o.FromEnvironment, "from-environment", "e", "", "Optional environment to use to obtain a version stream from, this overrides version-stream-git-url and version-stream-git-ref")

	return cmd, o
}

// Run implements the command
func (o *CLIOptions) Run() error {
	// upgrading to a specific version is not yet supported in brew so lets disable it for upgrades
	candidateInstallVersion, err := o.candidateInstallVersion()
	if err != nil {
		return errors.Wrapf(err, "failed to find jx cli version")
	}

	currentVersion, err := version.GetSemverVersion()
	if err != nil {
		return errors.Wrap(err, "failed to determine version of currently install jx release")
	}

	log.Logger().Debugf("Current version of jx: %s", util.ColorInfo(currentVersion))

	if o.needsUpgrade(currentVersion, candidateInstallVersion) {
		shouldUpgrade, err := o.ShouldUpdate(candidateInstallVersion)
		if err != nil {
			return errors.Wrap(err, "failed to determine if we should upgrade")
		}
		if shouldUpgrade {
			return o.InstallJx(true, candidateInstallVersion.String())
		}
	}

	return nil
}

func (o *CLIOptions) candidateInstallVersion() (semver.Version, error) {

	if o.Version == "" {
		gitURL := o.VersionStreamGitURL
		if o.FromEnvironment != "" {
			// get from the environment name
			jxClient, ns, err := jxclient.LazyCreateJXClientAndNamespace(o.JXClient, "")
			if err != nil {
				return semver.Version{}, err
			}
			envMap, _, err := kube.GetEnvironments(jxClient, ns)
			if err != nil {
				return semver.Version{}, errors.Wrapf(err, "failed to get Jenkins X environments when running in namespace %s", ns)
			}
			env := envMap[o.FromEnvironment]
			if env == nil {
				return semver.Version{}, errors.Errorf("no environment matching %s found", o.FromEnvironment)
			}
			gitURL = env.Spec.Source.URL
			if gitURL == "" {
				return semver.Version{}, errors.Errorf("no env.Spec.Source.URL to clone for environment %s", o.FromEnvironment)
			}
		}
		var err error
		o.Version, err = o.getJXVersion(gitURL)
		if err != nil {
			return semver.Version{}, errors.Wrapf(err, "failed to get jx cli version from %s", gitURL)
		}
	}

	requestedVersion, err := semver.New(o.Version)
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "invalid version requested: %s", o.Version)
	}
	return *requestedVersion, nil
}

func (o *CLIOptions) needsUpgrade(currentVersion, latestVersion semver.Version) bool {
	if latestVersion.EQ(currentVersion) {
		log.Logger().Infof("You are already on the latest version of jx %s", util.ColorInfo(currentVersion.String()))
		return false
	}
	return true
}

// ShouldUpdate checks if CLI version should be updated
func (o *CLIOptions) ShouldUpdate(newVersion semver.Version) (bool, error) {
	log.Logger().Debugf("Checking if should upgrade %s", newVersion)
	currentVersion, err := version.GetSemverVersion()
	if err != nil {
		return false, err
	}

	if newVersion.GT(currentVersion) {
		// Do not ask to update if we are using a dev build...
		for _, x := range currentVersion.Pre {
			if x.VersionStr == "dev" {
				log.Logger().Debugf("Ignoring possible update as it appears you are using a dev build - %s", currentVersion)
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

// InstallJx installs jx cli
func (o *CLIOptions) InstallJx(upgrade bool, version string) error {
	log.Logger().Debugf("installing jx %s", version)
	binary := "jx-cli"
	if !upgrade {
		flag, err := packages.ShouldInstallBinary(binary)
		if err != nil || !flag {
			return err
		}
	}

	extension := "tar.gz"
	if runtime.GOOS == "windows" {
		extension = "zip"
	}
	clientURL := fmt.Sprintf("%s%s/"+binary+"-%s-%s.%s", BinaryDownloadBaseURL, version, runtime.GOOS, runtime.GOARCH, extension)
	exe, err := os.Executable()
	if err != nil {
		return errors.Wrapf(err, "failed to get the jx executable which is running this command")
	}

	err = selfupdate.UpdateTo(clientURL, exe)
	if err != nil {
		return errors.Wrapf(err, "failed to upgrade jx cli to version %s", version)
	}
	log.Logger().Infof("Jenkins X client has been upgraded to version %s", version)
	return nil
}

func (o *CLIOptions) getJXVersion(gitURL string) (string, error) {
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}

	versionStreamDir, err := gitclient.CloneToDir(o.GitClient, gitURL, "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to clone git repo %s", gitURL)
	}
	// if o.FromEnvironment set move into the versionStream dir
	if o.FromEnvironment != "" {
		versionStreamDir = filepath.Join(versionStreamDir, "versionStream")
	}

	resolver := &versionstream.VersionResolver{
		VersionsDir: versionStreamDir,
	}

	data, err := resolver.StableVersion(versionstream.KindPackage, "jx-cli")
	if err != nil {
		return "", errors.Wrapf(err, "failed to get stable version for %s from versionstream %s", "jx-cli", gitURL)
	}
	return data.Version, nil
}
