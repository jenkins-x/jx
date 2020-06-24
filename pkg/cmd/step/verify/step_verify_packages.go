package verify

import (
	"fmt"
	"strings"

	options2 "github.com/jenkins-x/jx/v2/pkg/cmd/create/options"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/cmd/upgrade"
	"github.com/jenkins-x/jx/v2/pkg/config"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/jenkins-x/jx/v2/pkg/version"
	"github.com/jenkins-x/jx/v2/pkg/versionstream"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	verifyPackagesLong = templates.LongDesc(`
		Verifies the versions of the required command line packages


` + helper.SeeAlsoText("jx create project"))

	verifyPackagesExample = templates.Examples(`
		Verifies the versions of the required command line packages

		# verify packages and fail if any are not valid:
		jx step verify packages

		# override the error if the 'jx' binary is out of range (e.g. for development)
        export JX_DISABLE_VERIFY_JX="true"
		jx step verify packages
	`)
)

// StepVerifyPackagesOptions contains the command line flags
type StepVerifyPackagesOptions struct {
	step.StepOptions

	Namespace string
	HelmTLS   bool
	Packages  []string
	Dir       string
}

// NewCmdStepVerifyPackages creates the `jx step verify pod` command
func NewCmdStepVerifyPackages(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepVerifyPackagesOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "packages",
		Aliases: []string{"package"},
		Short:   "Verifies the versions of the required command line packages",
		Long:    verifyPackagesLong,
		Example: verifyPackagesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.HelmTLS, "helm-tls", "", false, "Whether to use TLS with helm")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to use to look for helm's tiller")
	cmd.Flags().StringArrayVarP(&options.Packages, "packages", "p", []string{"jx", "kubectl", "git", "helm", "kaniko"}, "The packages to verify")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to recursively look upwards for any 'jx-requirements.yml' file to determine the version stream")
	return cmd
}

// Run implements this command
func (o *StepVerifyPackagesOptions) Run() error {
	packages, table := o.GetPackageVersions(o.Namespace, o.HelmTLS)

	verifyMap := map[string]string{}
	for _, k := range o.Packages {
		verifyMap[k] = packages[k]
	}

	requirements, _, err := config.LoadRequirementsConfig(o.Dir, config.DefaultFailOnValidationError)
	if err != nil {
		return errors.Wrapf(err, "failed to load boot requirements")
	}
	vs := requirements.VersionStream
	u := vs.URL
	ref := vs.Ref
	log.Logger().Infof("Verifying the CLI packages using version stream URL: %s and git ref: %s\n", u, vs.Ref)

	resolver, err := o.CreateVersionResolver(u, ref)
	if err != nil {
		return errors.Wrapf(err, "failed to create version resolver")
	}

	// lets verify jx separately
	delete(packages, "jx")

	err = resolver.VerifyPackages(verifyMap)
	if err != nil {
		return err
	}
	err = o.verifyJXVersion(resolver)
	if err != nil {
		return err
	}

	log.Logger().Infof("CLI packages %s seem to be setup correctly", util.ColorInfo(strings.Join(o.Packages, ", ")))
	table.Render()

	return nil
}

func (o *StepVerifyPackagesOptions) verifyJXVersion(resolver versionstream.Streamer) error {
	currentVersion, err := version.GetSemverVersion()
	if err != nil {
		return errors.Wrap(err, "getting current jx version")
	}
	versionStreamVersion, err := o.GetLatestJXVersion(resolver)
	if err != nil {
		return errors.Wrap(err, "getting latest jx version")
	}
	info := util.ColorInfo
	latestVersionText := versionStreamVersion.String()

	if currentVersion.EQ(versionStreamVersion) {
		log.Logger().Infof("using version %s of %s", info(latestVersionText), info("jx"))
		return nil
	}

	// The case this happens is when we have a version stream ref which is outdated, should the version stream not get updated before the binary version is checked?
	if currentVersion.GE(versionStreamVersion) {
		log.Logger().Warnf("jx version specified in the version stream %s is %s. You are using %s. We highly recommend you upgrade versionstream ref", util.ColorInfo(resolver.GetVersionsDir()), util.ColorInfo(latestVersionText), util.ColorInfo(currentVersion.String()))
		return nil
	}

	log.Logger().Info("\n")
	log.Logger().Warnf("jx version specified in the version stream %s is %s. You are using %s. We highly recommend you upgrade to it.", util.ColorInfo(resolver.GetVersionsDir()), util.ColorInfo(latestVersionText), util.ColorInfo(currentVersion.String()))

	if o.BatchMode {
		log.Logger().Warnf("To upgrade to this new version use: %s then re-run %s", info("jx upgrade cli"), info("jx boot"))
		return nil
	}
	// skip checks for dev version
	if strings.Contains(currentVersion.String(), "-dev") {
		log.Logger().Warn("Skipping version upgrade since a dev version is used")
		return nil
	}

	// Todo: verify function should not upgrade, probably a different function is the right thing
	log.Logger().Info("\n")
	message := fmt.Sprintf("Would you like to upgrade to the %s version?", info("jx"))
	answer, err := util.Confirm(message, true, "Please indicate if you would like to upgrade the binary version.", o.GetIOFileHandles())
	if err != nil {
		return err
	}

	if answer {
		options := &upgrade.UpgradeCLIOptions{
			CreateOptions: options2.CreateOptions{
				CommonOptions: o.CommonOptions,
			},
		}
		options.Version = latestVersionText
		options.NoBrew = true
		err = options.Run()
		if err != nil {
			return err
		}
		log.Logger().Info("\n")
		log.Logger().Warnf("the version of %s has been updated to %s. Please re-run: %s", info("jx"), info(latestVersionText), info("jx boot"))
		log.Logger().Info("\n")

		return fmt.Errorf("the version of jx has been updated. Please re-run: jx boot")
	}

	return nil
}
