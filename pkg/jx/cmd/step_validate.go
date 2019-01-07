package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"k8s.io/apimachinery/pkg/util/errors"
)

const (
	optionMinJxVersion = "min-jx-version"
)

var (
	stepValidationOptions = []string{optionMinJxVersion}

	stepValidateLong = templates.LongDesc(`
		Validates the command line tools, container and platform to ensure a pipeline can run properly.

		This helps ensure that your platform installation, 'addons, builder images and Jenkinsfile' are all on compatible versions.
`)

	stepValidateExample = templates.Examples(`
		# Validates that the jx version is new enough
		jx validate --min-jx-version ` + version.VersionStringDefault(version.ExampleVersion) + `
			`)
)

// StepValidateOptions contains the command line flags
type StepValidateOptions struct {
	StepOptions

	MinimumJxVersion string
	Dir              string
}

// NewCmdStepValidate Creates a new Command object
func NewCmdStepValidate(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepValidateOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "validate",
		Short:   "Validates the command line tools, container and platform to ensure a pipeline can run properly",
		Long:    stepValidateLong,
		Example: stepValidateExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.MinimumJxVersion, optionMinJxVersion, "v", "", "The minimum version of the 'jx' command line tool required")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The project directory to look inside for the Project configuration for things like required addons")
	return cmd
}

// Run implements this command
func (o *StepValidateOptions) Run() error {
	errs := []error{}
	if o.MinimumJxVersion != "" {
		err := o.verifyJxVersion(o.MinimumJxVersion)
		if err != nil {
			errs = append(errs, err)
		}
	}
	errs = append(errs, o.verifyAddons()...)
	return errors.NewAggregate(errs)
}

func (o *StepValidateOptions) verifyJxVersion(minJxVersion string) error {
	require, err := semver.Parse(minJxVersion)
	if err != nil {
		return fmt.Errorf("Given jx version '%s' is not a valid semantic version: %s", minJxVersion, err)
	}
	current, err := version.GetSemverVersion()
	if err != nil {
		return fmt.Errorf("Could not find current jx version: %s", err)
	}
	if require.GT(current) {
		info := util.ColorInfo
		log.Infof("\nThe current installation of the %s CLI is too old: %s.\nWe require an installation of %s or later.\n\n", info("jx"), info(current.String()), info(require.String()))
		log.Infof(`To upgrade try these commands:

* to upgrade the platform:    %s
* to upgrade the CLI locally: %s

`, info("jx upgrade platform"), info("jx upgrade cli"))

		return fmt.Errorf("The current jx install is too old: %s. We require: %s or later", current.String(), require.String())
	}
	return nil
}

func (o *StepValidateOptions) verifyAddons() []error {
	errs := []error{}
	config, fileName, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		errs = append(errs, fmt.Errorf("Failed to load project config: %s", err))
		return errs
	}
	if len(config.Addons) == 0 {
		return errs
	}
	_, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	statusMap, err := o.Helm().StatusReleases(ns)
	if err != nil {
		errs = append(errs, fmt.Errorf("Failed to load addons statuses: %s", err))
		return errs
	}

	for _, addonConfig := range config.Addons {
		if addonConfig != nil {
			err := o.verifyAddon(addonConfig, fileName, statusMap)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

func (o *StepValidateOptions) verifyAddon(addonConfig *config.AddonConfig, fileName string, statusMap map[string]helm.Release) error {
	name := addonConfig.Name
	if name == "" {
		log.Warnf("Ignoring addon with no name inside the projects configuration file %s", fileName)
		return nil
	}
	ch := kube.AddonCharts[name]
	if ch == "" {
		return fmt.Errorf("No such addon name %s in %s: %s", name, fileName, util.InvalidArg(name, util.SortedMapKeys(kube.AddonCharts)))
	}
	status := statusMap[name].Status
	if status == "DEPLOYED" {
		return nil
	}
	info := util.ColorInfo

	log.Infof(`
The Project Configuration %s requires the %s addon to be installed. To fix this please type:

    %s

`, fileName, info(name), info(fmt.Sprintf("jx create addon %s", name)))

	return fmt.Errorf("The addon %s is required. Please install with: jx create addon %s", name, name)
}
