package git

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

// StepGitValidateOptions contains the command line flags
type StepGitValidateOptions struct {
	opts.StepOptions
}

var (
	stepGitValidateLong = templates.LongDesc(`
		This pipeline step validates that the .gitconfig is correctly configured

`)

	stepGitValidateExample = templates.Examples(`
		# validates the user.name & user.email values are set in the .gitconfig
		jx step git validate
`)
)

// NewCmdStepGitValidate creates a command to validate gitconfig
func NewCmdStepGitValidate(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepGitValidateOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "validate",
		Short:   "Validates the .gitconfig is correctly configured",
		Long:    stepGitValidateLong,
		Example: stepGitValidateExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	return cmd
}

// Run validates git config
func (o *StepGitValidateOptions) Run() error {
	// lets ignore errors which indicate no value set
	userName, _ := o.Git().Username("")
	userEmail, _ := o.Git().Email("")
	var err error
	if userName == "" {
		if !o.BatchMode {
			userName, err = util.PickValue("Please enter the name you wish to use with git: ", "", true, "", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
		if userName == "" {
			return fmt.Errorf("No Git user.name is defined. Please run the command: git config --global --add user.name \"MyName\"")
		}
		err = o.Git().SetUsername("", userName)
		if err != nil {
			return err
		}
	}
	if userEmail == "" {
		if !o.BatchMode {
			userEmail, err = util.PickValue("Please enter the email address you wish to use with git: ", "", true, "", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
		if userEmail == "" {
			return fmt.Errorf("No Git user.email is defined. Please run the command: git config --global --add user.email \"me@acme.com\"")
		}
		err = o.Git().SetEmail("", userEmail)
		if err != nil {
			return err
		}
	}
	log.Logger().Infof("Git configured for user: %s and email %s", util.ColorInfo(userName), util.ColorInfo(userEmail))
	return nil

}
