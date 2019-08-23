package pr

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/config"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/jenkins-x/jx/pkg/gits"

	"github.com/jenkins-x/jx/pkg/versionstream"

	"github.com/jenkins-x/jx/pkg/gits/operations"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cloud/gke"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	createVersionPullRequestLong = templates.LongDesc(`
		Creates a Pull Request on the versions git repository for a new versionstream of a chart/package
`)

	createVersionPullRequestExample = templates.Examples(`
		# create a Pull Request to update a chart versionstream
		jx step create pr versions -n jenkins-x/prow -v 1.2.3

		# create a Pull Request to update a chart versionstream to the latest found in the helm repo
		jx step create pr versions -n jenkins-x/prow 

		# create a Pull Request to update all charts matching a filter to the latest found in the helm repo
		jx step create pr versions pr -f "*"

		# create a Pull Request to update all charts in the 'jenkins-x' chart repository to the latest found in the helm repo
		jx step create pr versions -f "jenkins-x/*"

		# create a Pull Request to update all charts in the 'jenkins-x' chart repository and update the BDD test images
		jx step create pr versions -f "jenkins-x/*" --images

			`)
)

// StepCreatePullRequestVersionsOptions contains the command line flags
type StepCreatePullRequestVersionsOptions struct {
	StepCreatePrOptions

	Kind               string
	Name               string
	Includes           []string
	Excludes           []string
	UpdateTektonImages bool
}

// NewCmdStepCreateVersionPullRequest Creates a new Command object
func NewCmdStepCreateVersionPullRequest(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreatePullRequestVersionsOptions{
		StepCreatePrOptions: StepCreatePrOptions{
			StepCreateOptions: opts.StepCreateOptions{
				StepOptions: opts.StepOptions{
					CommonOptions: commonOpts,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "versions",
		Short:   "Creates a Pull Request on the versions git repository for a new versionstream of a chart/package",
		Long:    createVersionPullRequestLong,
		Example: createVersionPullRequestExample,
		Aliases: []string{"versionstream pullrequest", "versionstream"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "charts", "The kind of versionstream. Possible values: "+strings.Join(versionstream.KindStrings, ", "))
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "The name of the versionstream to update. e.g. the name of the chart like 'jenkins-x/prow'")
	cmd.Flags().StringArrayVarP(&options.Includes, "filter", "f", nil, "The name patterns to include - such as '*' for all names")
	cmd.Flags().StringArrayVarP(&options.Excludes, "excludes", "x", nil, "The name patterns to exclude")
	cmd.Flags().BoolVarP(&options.UpdateTektonImages, "images", "", false, "Update the tekton builder images for the Jenkins X Versions BDD tests")
	AddStepCreatePrFlags(cmd, &options.StepCreatePrOptions)
	return cmd
}

// ValidateVersionsOptions validates the common options for versionstream pr steps
func (o *StepCreatePullRequestVersionsOptions) ValidateVersionsOptions() error {
	if len(o.GitURLs) == 0 {
		// Default in the versions repo
		o.GitURLs = []string{config.DefaultVersionsURL}
	}
	if o.Kind == "" {
		return util.MissingOption("kind")
	}
	if util.StringArrayIndex(versionstream.KindStrings, o.Kind) < 0 {
		return util.InvalidOption("kind", o.Kind, versionstream.KindStrings)
	}

	// TODO really updating the images should be a different command
	if o.UpdateTektonImages && o.Name == "" && len(o.Includes) == 0 && o.Kind == "charts" {
		o.Kind = ""
	} else if len(o.Includes) == 0 {
		if o.Name == "" && !o.UpdateTektonImages {
			return util.MissingOption("name")
		}
	}
	return nil
}

// Run implements this command
func (o *StepCreatePullRequestVersionsOptions) Run() error {
	if err := o.ValidateVersionsOptions(); err != nil {
		return errors.WithStack(err)
	}

	modifyFns := make([]operations.ChangeFilesFn, 0)

	if o.UpdateTektonImages {

		builderImageVersion, err := findLatestBuilderImageVersion()
		if err != nil {
			return errors.WithStack(err)
		}

		log.Logger().Infof("the latest builder image version is %s\n", util.ColorInfo(builderImageVersion))
		pro := operations.PullRequestOperation{
			CommonOptions: o.CommonOptions,
			GitURLs:       o.GitURLs,
			SrcGitURL:     "https://github.com/jenkins-x/jenkins-x-builders.git",
			Base:          o.Base,
			BranchName:    o.BranchName,
			Version:       builderImageVersion,
			DryRun:        o.DryRun,
		}
		fn, err := operations.CreatePullRequestRegexFn(builderImageVersion, "gcr.io/jenkinsxio/builder-(?:maven|go|terraform):(?P<versions>.+)", "jenkins-x-*.yml")
		if err != nil {
			return errors.WithStack(err)
		}
		modifyFns = append(modifyFns, pro.WrapChangeFilesWithCommitFn("versions", fn))
		modifyFns = append(modifyFns, pro.WrapChangeFilesWithCommitFn("versions", operations.CreatePullRequestBuildersFn(builderImageVersion)))
	}
	if len(o.Includes) > 0 {
		switch versionstream.VersionKind(o.Kind) {
		case versionstream.KindChart:
			modifyFns = append(modifyFns, o.CreatePullRequestUpdateVersionFilesFn(o.Includes, o.Excludes, o.Kind, o.Helm()))
		default:
			return fmt.Errorf("we do not yet support finding the latest version of kind %s", o.Kind)
		}
	} else {
		pro := operations.PullRequestOperation{
			CommonOptions: o.CommonOptions,
			GitURLs:       o.GitURLs,
			Base:          o.Base,
			BranchName:    o.BranchName,
			Version:       o.Version,
			DryRun:        o.DryRun,
		}
		vaultClient, err := o.SystemVaultClient("")
		if err != nil {
			vaultClient = nil
		}
		modifyFns = append(modifyFns, pro.WrapChangeFilesWithCommitFn("versions", operations.CreateChartChangeFilesFn(o.Name, o.Version, o.Kind, &pro, o.Helm(), vaultClient, o.In, o.Out, o.Err)))
	}

	o.SrcGitURL = ""    // there is no src url for the overall PR
	o.SkipCommit = true // As we've done all the commits already
	return o.CreatePullRequest("versionstream", func(dir string, gitInfo *gits.GitRepository) ([]string, error) {
		if versionstream.VersionKind(o.Kind) == versionstream.KindChart {
			_, err := o.HelmInitDependency(dir, o.DefaultReleaseCharts())
			if err != nil {
				return nil, errors.Wrap(err, "failed to ensure the helm repositories were setup")
			}
		}
		log.Logger().Info("updating helm repositories to find the latest chart versions...\n")
		err := o.Helm().UpdateRepo()
		if err != nil {
			return nil, errors.Wrap(err, "failed to update helm repos")
		}
		for _, fn := range modifyFns {
			// in a versions PR there is no overall "old version" - it's done commit by commit
			_, err := fn(dir, gitInfo)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
		return nil, nil
	})
}

func findLatestBuilderImageVersion() (string, error) {
	cmd := util.Command{
		Name: "gcloud",
		Args: []string{
			"container", "images", "list-tags", "gcr.io/jenkinsxio/builder-maven", "--format", "json",
		},
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return gke.FindLatestImageTag(output)
}

//CreatePullRequestUpdateVersionFilesFn creates the ChangeFilesFn for directory tree of stable version files, applying the includes and excludes
func (o *StepCreatePullRequestVersionsOptions) CreatePullRequestUpdateVersionFilesFn(includes []string, excludes []string, kindStr string, helmer helm.Helmer) operations.ChangeFilesFn {

	return func(dir string, gitInfo *gits.GitRepository) (i []string, e error) {
		answer := make([]string, 0)

		kindDir := filepath.Join(dir, kindStr)
		glob := filepath.Join(kindDir, "*", "*.yml")
		paths, err := filepath.Glob(glob)
		if err != nil {
			return nil, errors.Wrapf(err, "bad glob pattern %s", glob)
		}
		for _, path := range paths {
			name, err := filepath.Rel(kindDir, path)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract base path from %s", path)
			}
			ext := filepath.Ext(name)
			if ext != "" {
				name = strings.TrimSuffix(name, ext)
			}
			if !util.StringMatchesAny(name, includes, excludes) {
				continue
			} else {
				pro := operations.PullRequestOperation{
					CommonOptions: o.CommonOptions,
					GitURLs:       o.GitURLs,
					Base:          o.Base,
					BranchName:    o.BranchName,
					DryRun:        o.DryRun,
				}
				vaultClient, err := o.SystemVaultClient("")
				if err != nil {
					vaultClient = nil
				}
				cff := pro.WrapChangeFilesWithCommitFn(kindStr, operations.CreateChartChangeFilesFn(name, "", kindStr, &pro, o.Helm(), vaultClient, o.In, o.Out, o.Err))
				a, err := cff(dir, gitInfo)
				if err != nil {
					if isFailedToFindLatestChart(err) {
						log.Logger().Warnf("Failed to find latest chart for %s", util.ColorInfo(name))
					} else {
						return nil, errors.WithStack(err)
					}
				}
				answer = append(answer, a...)
			}
		}
		return answer, nil
	}
}

func isFailedToFindLatestChart(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "failed to find latest chart version for ")
}
