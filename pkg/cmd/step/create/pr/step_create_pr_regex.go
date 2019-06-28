package pr

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

var (
	createPullRequestRegexLong = templates.LongDesc(`
		Creates a Pull Request on a git repository updating the requirements.yaml and values.yaml with the new version
`)

	createPullRequestRegexExample = templates.Examples(`
					`)
)

// StepCreatePullRequestRegexOptions contains the command line flags
type StepCreatePullRequestRegexOptions struct {
	StepCreatePrOptions

	Regexp string
	Files  string
}

// NewCmdStepCreatePullRequestRegex Creates a new Command object
func NewCmdStepCreatePullRequestRegex(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreatePullRequestRegexOptions{
		StepCreatePrOptions: StepCreatePrOptions{
			StepCreateOptions: opts.StepCreateOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: commonOpts,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "regex",
		Short:   "Creates a Pull Request on a git repository, doing an update using the provided regex",
		Long:    createPullRequestRegexLong,
		Example: createPullRequestRegexExample,
		Aliases: []string{"version pullrequest"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	AddStepCreatePrFlags(cmd, &options.StepCreatePrOptions)
	cmd.Flags().StringVarP(&options.Regexp, "regex", "", "", "The regex to use when doing updates")
	cmd.Flags().StringVarP(&options.Files, "files", "", "", "A glob describing the files to change")
	return cmd
}

// Run implements this command
func (o *StepCreatePullRequestRegexOptions) Run() error {
	if err := o.ValidateOptions(); err != nil {
		return errors.WithStack(err)
	}
	if o.Regexp == "" {
		return util.MissingOption("regex")
	}
	// ensure the regexp is multi-line
	if !strings.HasPrefix(o.Regexp, "(?m") {
		o.Regexp = fmt.Sprintf("(?m:%s)", o.Regexp)
	}
	regexp, err := regexp.Compile(o.Regexp)
	if err != nil {
		return errors.Wrapf(err, "%s does not compile", o.Regexp)
	}
	if o.Version == "" {
		return util.MissingOption("version")
	}
	if o.SrcGitURL == "" {
		log.Logger().Warnf("srcRepo is not provided so generated PR will not be correctly linked in release notesPR")
	}
	err = o.CreatePullRequest("regex",
		func(dir string, gitInfo *gits.GitRepository) ([]string, error) {
			oldVersions := make([]string, 0)
			matches, err := filepath.Glob(filepath.Join(dir, o.Files))
			if err != nil {
				return nil, errors.Wrapf(err, "applying glob %s", o.Files)
			}

			// iterate over the glob matches
			for _, path := range matches {

				data, err := ioutil.ReadFile(path)
				if err != nil {
					return nil, errors.Wrapf(err, "reading %s", path)
				}
				info, err := os.Stat(path)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				s := string(data)
				answer := util.ReplaceAllStringSubmatchFunc(regexp, s, func(groups []util.Group) []string {
					answer := make([]string, 0)
					for _, group := range groups {
						oldVersions = append(oldVersions, group.Value)
						answer = append(answer, o.Version)
					}
					return answer
				})
				err = ioutil.WriteFile(path, []byte(answer), info.Mode())
				if err != nil {
					return nil, errors.Wrapf(err, "writing %s", path)
				}
			}
			if err != nil {
				return nil, errors.WithStack(err)
			}
			return oldVersions, nil
		})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
