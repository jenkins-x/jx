package pr

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	createtPullRequestGoLong = templates.LongDesc(`
		Creates a Pull Request to change a go module dependency, updating the go.mod and go.sum files to use a new version

		Files named Makefile or Makefile.* will be updated
`)

	createtPullRequestGoExample = templates.Examples(`
		# update a go dependency 
		jx step create pr go --name github.com/myorg/myrepo --version v1.2.3 --repo https://github.com/jenkins-x/cloud-environments.git

		# update a go dependency using a custom build step (to update the 'go.sum' file) 
		jx step create pr go --name github.com/myorg/myrepo --version v1.2.3 --build "make something" --repo https://github.com/jenkins-x/cloud-environments.git
					`)
)

// StepCreatetPullRequestGoOptions contains the command line flags
type StepCreatetPullRequestGoOptions struct {
	StepCreatePrOptions

	Name         string
	BuildCommand string
}

// NewCmdStepCreatetPullRequestGo Creates a new Command object
func NewCmdStepCreatetPullRequestGo(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreatetPullRequestGoOptions{
		StepCreatePrOptions: StepCreatePrOptions{
			StepCreateOptions: opts.StepCreateOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: commonOpts,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "go",
		Short:   "Creates a Pull Request on a git repository updating a go module dependency",
		Long:    createtPullRequestGoLong,
		Example: createtPullRequestGoExample,
		Aliases: []string{"golang"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	AddStepCreatePrFlags(cmd, &options.StepCreatePrOptions)
	cmd.Flags().StringVarP(&options.Name, "name", "", "", "The name of the go module dependency to use when doing updates")
	cmd.Flags().StringVarP(&options.BuildCommand, "build", "", "make build", "The build command to update the 'go.sum' file after the change to the source")
	return cmd
}

// ValidateGoOptions validates the common options for make pr steps
func (o *StepCreatetPullRequestGoOptions) ValidateGoOptions() error {
	if err := o.ValidateOptions(); err != nil {
		return errors.WithStack(err)
	}
	if o.Name == "" {
		return util.MissingOption("name")
	}
	if o.SrcGitURL == "" {
		log.Logger().Warnf("srcRepo is not provided so generated PR will not be correctly linked in release notesPR")
	}

	return nil
}

// Run implements this command
func (o *StepCreatetPullRequestGoOptions) Run() error {
	if err := o.ValidateGoOptions(); err != nil {
		return errors.WithStack(err)
	}
	ro := StepCreatePullRequestRegexOptions{
		StepCreatePrOptions: o.StepCreatePrOptions,
		Files:               []string{"go.mod"},
		Regexp:              fmt.Sprintf(`^\s*\Q%s\E\s+(?P<version>.+)`, o.Name),
		Kind:                "go",
		PostChangeCallback: func(dir string, gitInfo *gits.GitRepository) error {
			return o.runGoBuild(dir)
		},
	}
	err := ro.Run()
	if err != nil {
		return errors.Wrapf(err, "executing regex %s on globs %+v", ro.Regexp, ro.Files)
	}
	return nil
}

func (o *StepCreatetPullRequestGoOptions) runGoBuild(dir string) error {
	build := o.BuildCommand
	if build == "" {
		log.Logger().Warn("no build command so we will not change the 'go.sum' command")
		return nil
	}
	log.Logger().Infof("running the build command: %s in the directory %s to update the 'go.sum' file\n", util.ColorInfo(build), dir)

	values := strings.Split(build, " ")
	err := o.RunCommandVerboseAt(dir, values[0], values[1:]...)
	log.Logger().Infof("")
	return err
}
