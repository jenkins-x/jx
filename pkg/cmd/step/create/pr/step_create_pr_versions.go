package pr

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cloud/gke"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var (
	createVersionPullRequestLong = templates.LongDesc(`
		Creates a Pull Request on the versions git repository for a new version of a chart/package
`)

	createVersionPullRequestExample = templates.Examples(`
		# create a Pull Request to update a chart version
		jx step create pr versions -n jenkins-x/prow -v 1.2.3

		# create a Pull Request to update a chart version to the latest found in the helm repo
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
	opts.StepOptions

	Kind               string
	Name               string
	Includes           []string
	Excludes           []string
	Version            string
	UpdateTektonImages bool

	updatedHelmRepo     bool
	builderImageVersion string

	PullRequestDetails opts.PullRequestDetails
}

// StepCreateVersionPullRequestResults stores the generated results
type StepCreatePullRequestVersionsResults struct {
	Pipeline    *pipelineapi.Pipeline
	Task        *pipelineapi.Task
	PipelineRun *pipelineapi.PipelineRun
}

// NewCmdStepCreateVersionPullRequest Creates a new Command object
func NewCmdStepCreateVersionPullRequest(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreatePullRequestVersionsOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "versions",
		Short:   "Creates a Pull Request on the versions git repository for a new version of a chart/package",
		Long:    createVersionPullRequestLong,
		Example: createVersionPullRequestExample,
		Aliases: []string{"version pullrequest"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.PullRequestDetails.RepositoryGitURL, "repo", "r", opts.DefaultVersionsURL, "Jenkins X versions Git repo")
	cmd.Flags().StringVarP(&options.PullRequestDetails.RepositoryBranch, "branch", "", "master", "the versions git repository branch to clone and generate a pull request from")
	cmd.Flags().StringVarP(&options.Kind, "kind", "k", "charts", "The kind of version. Possible values: "+strings.Join(version.KindStrings, ", "))
	cmd.Flags().StringVarP(&options.Name, "name", "n", "", "The name of the version to update. e.g. the name of the chart like 'jenkins-x/prow'")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The version to change. If no version is supplied the latest version is found")
	cmd.Flags().StringArrayVarP(&options.Includes, "filter", "f", nil, "The name patterns to include - such as '*' for all names")
	cmd.Flags().StringArrayVarP(&options.Excludes, "excludes", "x", nil, "The name patterns to exclude")
	cmd.Flags().BoolVarP(&options.UpdateTektonImages, "images", "", false, "Update the tekton builder images for the Jenkins X Versions BDD tests")
	return cmd
}

// Run implements this command
func (o *StepCreatePullRequestVersionsOptions) Run() error {
	if o.Kind == "" {
		return util.MissingOption("kind")
	}
	if util.StringArrayIndex(version.KindStrings, o.Kind) < 0 {
		return util.InvalidOption("kind", o.Kind, version.KindStrings)
	}
	opts := &o.PullRequestDetails

	if opts.RepositoryGitURL == "" {
		return util.MissingOption("repo")
	}
	dir, err := ioutil.TempDir("", "create-version-pr")
	if err != nil {
		return err
	}

	if o.UpdateTektonImages {
		o.builderImageVersion, err = o.findLatestBuilderImageVersion()
		if err != nil {
			return err
		}

		log.Logger().Infof("the latest builder image version is %s\n", util.ColorInfo(o.builderImageVersion))
	}

	if len(o.Includes) == 0 {
		if o.Name == "" {
			return util.MissingOption("name")
		}

		if o.Version == "" && o.Kind == string(version.KindChart) {
			o.Version, err = o.findLatestChartVersion(o.Name)
			if err != nil {
				return errors.Wrapf(err, "failed to find latest chart version for %s", o.Name)
			}
			log.Logger().Infof("found latest version %s for chart %s\n", util.ColorInfo(o.Version), util.ColorInfo(o.Name))
		}
		if o.Version == "" {
			return util.MissingOption("version")
		}
		o.PullRequestDetails.BranchNameText = strings.Replace("upgrade-"+o.Name+"-"+o.Version, "/", "-", -1)
		o.PullRequestDetails.BranchNameText = strings.Replace(o.PullRequestDetails.BranchNameText, ".", "-", -1)

		o.PullRequestDetails.Title = fmt.Sprintf("%s version upgrade of %s", o.Kind, o.Name)
		o.PullRequestDetails.Message = fmt.Sprintf("change %s to version %s", o.Name, o.Version)
	} else {
		o.PullRequestDetails.BranchNameText = "upgrade-chart-versions-" + string(uuid.NewUUID())
		o.PullRequestDetails.Title = "upgrade chart versions"
		o.PullRequestDetails.Message = fmt.Sprintf("change %s to version %s", o.Name, o.Version)
	}

	opts.Dir = dir
	opts.RepositoryMessage = "versions repository"

	return o.CreatePullRequest(&o.PullRequestDetails, func() error {
		return o.modifyFiles(dir)
	})
}

func (o *StepCreatePullRequestVersionsOptions) modifyFiles(dir string) error {
	if version.VersionKind(o.Kind) == version.KindChart {
		err := o.ensureHelmReposSetup(dir)
		if err != nil {
			return err
		}
	}

	if o.builderImageVersion != "" {
		err := o.modifyRegex(filepath.Join(dir, "jenkins-x-*.yml"), "gcr.io/jenkinsxio/builder-go-maven:(.+)", "gcr.io/jenkinsxio/builder-go-maven:"+o.builderImageVersion)
		if err != nil {
			return errors.Wrap(err, "modifying the BDD test version YAMLs (builder-go-maven)")
		}
		err = o.modifyRegex(filepath.Join(dir, "jenkins-x-*.yml"), "gcr.io/jenkinsxio/builder-go:(.+)", "gcr.io/jenkinsxio/builder-go:"+o.builderImageVersion)
		if err != nil {
			return errors.Wrap(err, "modifying the BDD test version YAMLs (builder-go)")
		}
		err = o.modifyRegex(filepath.Join(dir, "jenkins-x-*.yml"), "gcr.io/jenkinsxio/builder-terraform:(.+)", "gcr.io/jenkinsxio/builder-terraform:"+o.builderImageVersion)
		if err != nil {
			return errors.Wrap(err, "modifying the BDD test version YAMLs (builder-terraform)")
		}

		pattern := filepath.Join(dir, string(version.KindDocker), "gcr.io", "jenkinsxio", "builder-*.yml")
		err = o.modifyVersionYamlFiles(pattern, o.builderImageVersion, "builder-base.yml")
		if err != nil {
			return errors.Wrap(err, "modifying the builder image versions")
		}
	}
	if len(o.Includes) > 0 {
		switch version.VersionKind(o.Kind) {
		case version.KindChart:
			return o.findLatestChartVersions(dir)
		default:
			return fmt.Errorf("We do not yet support finding the latest version of kind %s", o.Kind)
		}
	}

	kind := version.VersionKind(o.Kind)
	data, err := version.LoadStableVersion(dir, kind, o.Name)
	if err != nil {
		return err
	}
	if data.Version == o.Version {
		return nil
	}
	data.Version = o.Version
	err = version.SaveStableVersion(dir, kind, o.Name, data)
	if err != nil {
		return errors.Wrapf(err, "failed to save version file")
	}
	return nil
}

func (o *StepCreatePullRequestVersionsOptions) findLatestChartVersions(dir string) error {
	callback := func(kind version.VersionKind, name string, stableVersion *version.StableVersion) (bool, error) {
		if !util.StringMatchesAny(name, o.Includes, o.Excludes) {
			return true, nil
		}
		v, err := o.findLatestChartVersion(name)
		if err != nil {
			log.Logger().Warnf("failed to find latest version of %s: %s\n", name, err.Error())
			return true, nil
		}
		if v != stableVersion.Version {
			stableVersion.Version = v
			err = version.SaveStableVersion(dir, kind, name, stableVersion)
			if err != nil {
				return false, err
			}
			o.PullRequestDetails.Message += fmt.Sprintf("change `%s` to version `%s`\n", name, v)
		}
		return true, nil
	}

	err := version.ForEachKindVersion(dir, version.VersionKind(o.Kind), callback)
	return err
}

func (o *StepCreatePullRequestVersionsOptions) modifyVersionYamlFiles(globPattern string, newVersion string, excludeFiles ...string) error {
	files, err := filepath.Glob(globPattern)
	if err != nil {
		return errors.Wrapf(err, "failed to create glob from pattern %s", globPattern)
	}

	for _, path := range files {
		_, name := filepath.Split(path)
		if util.StringArrayIndex(excludeFiles, name) >= 0 {
			continue
		}
		data, err := version.LoadStableVersionFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load version info for %s", path)
		}
		if data.Version == "" || data.Version == o.Version {
			continue
		}
		data.Version = newVersion
		err = version.SaveStableVersionFile(path, data)
		if err != nil {
			return errors.Wrapf(err, "failed to save version info for %s", path)
		}
	}
	return nil
}

func (o *StepCreatePullRequestVersionsOptions) findLatestChartVersion(name string) (string, error) {
	err := o.updateHelmRepo()
	if err != nil {
		return "", err
	}
	info, err := o.Helm().SearchChartVersions(name)
	if err != nil {
		return "", err
	}
	if len(info) == 0 {
		return "", fmt.Errorf("no version found for chart %s", name)
	}
	if o.Verbose {
		log.Logger().Infof("found %d versions:\n", len(info))
		for _, v := range info {
			log.Logger().Infof("    %s:\n", v)
		}
	}
	return info[0], nil
}

// updateHelmRepo updates the helm repos if required
func (o *StepCreatePullRequestVersionsOptions) updateHelmRepo() error {
	if o.updatedHelmRepo {
		return nil
	}
	log.Logger().Info("updating helm repositories to find the latest chart versions...\n")
	err := o.Helm().UpdateRepo()
	if err != nil {
		return errors.Wrap(err, "failed to update helm repos")
	}
	o.updatedHelmRepo = true
	return nil
}

func (o *StepCreatePullRequestVersionsOptions) findLatestBuilderImageVersion() (string, error) {
	output, err := o.GetCommandOutput("", "gcloud", "container", "images", "list-tags", "gcr.io/jenkinsxio/builder-maven", "--format", "json")
	if err != nil {
		return "", err
	}
	return gke.FindLatestImageTag(output)
}

// modifyRegex performs a search and replace of the given regular expression with the replacement in the given set of globPattern files
func (o *StepCreatePullRequestVersionsOptions) modifyRegex(globPattern string, regexPattern string, replacement string) error {
	files, err := filepath.Glob(globPattern)
	if err != nil {
		return errors.Wrapf(err, "failed to find glob pattern %s", globPattern)
	}
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return errors.Wrapf(err, "failed to parse regex %s", regexPattern)
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", file)
		}
		data = re.ReplaceAll(data, []byte(replacement))
		err = ioutil.WriteFile(file, data, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save file %s", file)
		}
	}
	return nil
}

func (o *StepCreatePullRequestVersionsOptions) ensureHelmReposSetup(dir string) error {
	_, err := o.HelmInitDependency(dir, o.DefaultReleaseCharts())
	return errors.Wrap(err, "failed to ensure the helm repositories were setup")
}
