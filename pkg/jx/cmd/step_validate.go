package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
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
}

// NewCmdStepValidate Creates a new Command object
func NewCmdStepValidate(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StepValidateOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
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
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.MinimumJxVersion, optionMinJxVersion, "v", "", "The minimum version of the 'jx' command line tool required")
	return cmd
}

// Run implements this command
func (o *StepValidateOptions) Run() error {
	var err error
	count := 0
	if o.MinimumJxVersion != "" {
		err = o.verifyJxVersion(o.MinimumJxVersion)
		count++
	}
	if count == 0 {
		return fmt.Errorf("No validation options supplied. Please use one of: --%s\n", strings.Join(stepValidationOptions, ", --"))
	}
	return err
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
		o.Printf("\nThe current installation of the %s CLI is too old: %s.\nWe require an installation of %s or later.\n\n", info("jx"), info(current.String()), info(require.String()))
		o.Printf(`To upgrade try these commands:

* to upgrade the platform:    %s
* to upgrade the CLI locally: %s

`, info("jx upgrade platform"), info("jx upgrade cli"))

		return fmt.Errorf("The current jx install is too old: %s. We require: %s or later", current.String(), require.String())
	}
	return nil
}
