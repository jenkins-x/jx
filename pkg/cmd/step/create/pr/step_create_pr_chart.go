package pr

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

var (
	createPullRequestChartLong = templates.LongDesc(`
		Creates a Pull Request on a git repository updating the requirements.yaml and values.yaml with the new version
`)

	createPullRequestChartExample = templates.Examples(`
					`)
)

// StepCreatePullRequestChartsOptions contains the command line flags
type StepCreatePullRequestChartsOptions struct {
	StepCreatePrOptions

	Version string
	Name    string
}

// StepCreatePullRequestChartResults stores the generated results
type StepCreatePullRequestChartResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
}

// NewCmdStepCreatePullRequestChart Creates a new Command object
func NewCmdStepCreatePullRequestChart(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreatePullRequestChartsOptions{
		StepCreatePrOptions: StepCreatePrOptions{
			StepCreateOptions: opts.StepCreateOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: commonOpts,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "chart",
		Short:   "Creates a Pull Request on a git repository updating the Chart",
		Long:    createPullRequestChartLong,
		Example: createPullRequestChartExample,
		Aliases: []string{"version pullrequest"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	AddStepCreatePrFlags(cmd, &options.StepCreatePrOptions)
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "The name of the property to update")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The version to change. If no version is supplied the latest version is found")
	return cmd
}

// Run implements this command
func (o *StepCreatePullRequestChartsOptions) Run() error {
	if err := o.ValidateOptions(); err != nil {
		return errors.WithStack(err)
	}
	if o.Name == "" {
		return util.MissingOption("name")
	}
	if o.Version == "" {
		return util.MissingOption("version")
	}
	if o.SrcGitURL == "" {
		log.Logger().Warnf("srcRepo is not provided so generated PR will not be correctly linked in release notesPR")
	}
	err := o.CreatePullRequest(
		func(dir string, gitInfo *gits.GitRepository) (s string, details *gits.PullRequestDetails, e error) {
			oldVersions := make([]string, 0)
			// walk the filepath, looking for values.yaml and requirements.yaml
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				base := filepath.Base(path)
				if base == helm.RequirementsFileName {
					requirements, err := helm.LoadRequirementsFile(path)
					if err != nil {
						return errors.Wrapf(err, "loading %s", path)
					}
					oldVersions = append(oldVersions, helm.UpdateRequirementsToNewVersion(requirements, o.Name, o.Version)...)
					err = helm.SaveFile(path, *requirements)
					if err != nil {
						return errors.Wrapf(err, "saving %s", path)
					}
				} else if base == helm.ValuesFileName {
					values, err := ioutil.ReadFile(path)
					if err != nil {
						return errors.Wrapf(err, "reading %s", path)
					}
					values, moreOldVersions := helm.UpdateImagesInValuesToNewVersion(values, o.Name, o.Version)
					oldVersions = append(oldVersions, moreOldVersions...)
					err = ioutil.WriteFile(path, values, info.Mode())
					if err != nil {
						return errors.Wrapf(err, "writing %s", path)
					}
				}
				return nil
			})
			if err != nil {
				return "", nil, errors.WithStack(err)
			}
			commitMessage, details, err := o.CreateDependencyUpdatePRDetails("chart", o.SrcGitURL, gitInfo, strings.Join(oldVersions, ", "), o.Version, o.Component)
			if err != nil {
				return "", nil, errors.WithStack(err)
			}
			return commitMessage, details, nil
		})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
