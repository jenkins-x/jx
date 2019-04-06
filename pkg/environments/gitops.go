package environments

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"

	jenkinsio "github.com/jenkins-x/jx/pkg/apis/jenkins.io"

	"github.com/ghodss/yaml"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"k8s.io/helm/pkg/proto/hapi/chart"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
)

// PullRequestDetails is the details for creating a pull request
type PullRequestDetails struct {
	Message    string
	BranchName string
	Title      string
}

//ValuesFiles is a wrapper for a slice of values files to allow them to be passed around as a pointer
type ValuesFiles struct {
	Items []string
}

// ModifyChartFn callback for modifying a chart, requirements, the chart metadata,
// the values.yaml and all files in templates are unmarshaled, and the root dir for the chart is passed
type ModifyChartFn func(requirements *helm.Requirements, metadata *chart.Metadata, existingValues map[string]interface{},
	templates map[string]string, dir string, pullRequestDetails *PullRequestDetails) error

// ConfigureGitFn callback to optionally configure git before its used for creating commits and PRs
type ConfigureGitFn func(dir string, gitInfo *gits.GitRepository, gitAdapter gits.Gitter) error

// EnvironmentPullRequestOptions are options for creating a pull request against an environment.
// The provide a Gitter client for performing git operations, a GitProvider client for talking to the git provider,
// a callback ModifyChartFn which is where the changes you want to make are defined,
// and a ConfigureGitFn which is run allowing you to add external git configuration.
type EnvironmentPullRequestOptions struct {
	Gitter        gits.Gitter
	GitProvider   gits.GitProvider
	ModifyChartFn ModifyChartFn
	ConfigGitFn   ConfigureGitFn
}

// Create a pull request against the environment repository for env.
// The EnvironmentPullRequestOptions are used to provide a Gitter client for performing git operations,
// a GitProvider client for talking to the git provider,
// a callback ModifyChartFn which is where the changes you want to make are defined,
// and a ConfigureGitFn which is run allowing you to add external git configuration.
// The branchNameText defines the branch name used, the title is used for both the commit and the pull request title,
// the message as the body for both the commit and the pull request,
// and the pullRequestInfo for any existing PR that exists to modify the environment that we want to merge these
// changes into.
func (o *EnvironmentPullRequestOptions) Create(env *jenkinsv1.Environment, environmentsDir string,
	pullRequestDetails *PullRequestDetails, pullRequestInfo *gits.PullRequestInfo) (*gits.PullRequestInfo, error) {

	dir, base, gitInfo, fork, err := o.PullEnvironmentRepo(env, environmentsDir)
	if err != nil {
		return nil, errors.Wrapf(err, "pulling environment repo %s into %s", env.Spec.Source.URL,
			environmentsDir)
	}

	branchName := o.Gitter.ConvertToValidBranchName(pullRequestDetails.BranchName)

	branchNames, err := o.Gitter.RemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load remote branch names")
	}
	//log.Infof("Found remote branch names %s\n", strings.Join(branchNames, ", "))
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchNameUUID, err := uuid.NewV4()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		branchName += "-" + branchNameUUID.String()
	}
	err = o.Gitter.CreateBranch(dir, branchName)
	if err != nil {
		return nil, err
	}
	err = o.Gitter.Checkout(dir, branchName)
	if err != nil {
		return nil, err
	}

	err = ModifyChartFiles(dir, pullRequestDetails, o.ModifyChartFn)
	if err != nil {
		return nil, err
	}
	return o.PushEnvironmentRepo(dir, branchName, gitInfo, base, pullRequestDetails, pullRequestInfo, fork)
}

// PushEnvironmentRepo commits and pushes the changes in the repo rooted at dir.
// It creates a branch called branchName from a base.
// It uses the pullRequestDetails for the message and title for the commit and PR.
// It uses and updates pullRequestInfo to identify whether to rebase an existing PR.
func (o *EnvironmentPullRequestOptions) PushEnvironmentRepo(dir string, branchName string,
	gitInfo *gits.GitRepository, base string, pullRequestDetails *PullRequestDetails,
	pullRequestInfo *gits.PullRequestInfo, fork bool) (*gits.PullRequestInfo, error) {
	err := o.Gitter.Add(dir, "-A")
	if err != nil {
		return nil, err
	}
	changed, err := o.Gitter.HasChanges(dir)
	if err != nil {
		return nil, err
	}
	if !changed {
		log.Warnf("%s\n", "No changes made to the GitOps Environment source code. Code must be up to date!")
		return nil, nil
	}
	err = o.Gitter.CommitDir(dir, pullRequestDetails.Message)
	if err != nil {
		return nil, err
	}
	// lets rebase an existing PR
	if pullRequestInfo != nil && pullRequestInfo.PullRequestArguments.Head != "" {
		err = o.Gitter.ForcePushBranch(dir, branchName, pullRequestInfo.PullRequestArguments.Head)
		if err != nil {
			return nil, errors.Wrapf(err, "rebasing existing PR on %s", pullRequestInfo.PullRequestArguments.Head)
		}
	}

	err = o.Gitter.Push(dir)
	if err != nil {
		return nil, err
	}

	headPrefix := ""

	username := o.GitProvider.CurrentUsername()
	if username == "" {
		return nil, fmt.Errorf("no git user name found")
	}
	if gitInfo.Organisation != username && fork {
		headPrefix = username + ":"
	}

	gha := &gits.GitPullRequestArguments{
		GitRepository: gitInfo,
		Title:         pullRequestDetails.Title,
		Body:          pullRequestDetails.Message,
		Base:          base,
		Head:          headPrefix + branchName,
	}

	pr, err := o.GitProvider.CreatePullRequest(gha)
	if err != nil {
		return nil, err
	}
	log.Infof("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return &gits.PullRequestInfo{
		GitProvider:          o.GitProvider,
		PullRequest:          pr,
		PullRequestArguments: gha,
	}, nil
}

// ModifyChartFiles modifies the chart files in the given directory using the given modify function
func ModifyChartFiles(dir string, details *PullRequestDetails, modifyFn ModifyChartFn) error {
	requirementsFile, err := helm.FindRequirementsFileName(dir)
	if err != nil {
		return err
	}
	requirements, err := helm.LoadRequirementsFile(requirementsFile)
	if err != nil {
		return err
	}

	chartFile, err := helm.FindChartFileName(dir)
	if err != nil {
		return err
	}
	chart, err := helm.LoadChartFile(chartFile)
	if err != nil {
		return err
	}

	valuesFile, err := helm.FindValuesFileName(dir)
	if err != nil {
		return err
	}
	values, err := helm.LoadValuesFile(valuesFile)
	if err != nil {
		return err
	}
	templatesDir, err := helm.FindTemplatesDirName(dir)
	if err != nil {
		return err
	}
	templates, err := helm.LoadTemplatesDir(templatesDir)
	if err != nil {
		return err
	}

	// lets pass in the folder containing the `Chart.yaml` which is the `env` dir in GitOps management
	chartDir, _ := filepath.Split(chartFile)

	err = modifyFn(requirements, chart, values, templates, chartDir, details)
	if err != nil {
		return err
	}

	err = helm.SaveFile(requirementsFile, requirements)
	if err != nil {
		return err
	}

	err = helm.SaveFile(chartFile, chart)
	if err != nil {
		return err
	}
	return nil
}

// PullEnvironmentRepo pulls the repo for env into environmentsDir
func (o *EnvironmentPullRequestOptions) PullEnvironmentRepo(env *jenkinsv1.Environment,
	environmentsDir string) (string, string, *gits.GitRepository, bool, error) {
	source := &env.Spec.Source
	gitURL := source.URL
	fork := false
	if gitURL == "" {
		return "", "", nil, fork, fmt.Errorf("No source git URL")
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return "", "", nil, fork, errors.Wrapf(err, "failed to parse git URL %s", gitURL)
	}

	username := ""
	userDetails := auth.UserAuth{}
	originalOrg := gitInfo.Organisation
	originalRepo := gitInfo.Name

	provider := o.GitProvider
	git := o.Gitter

	if o.GitProvider == nil {
		log.Warnf("No GitProvider specified!\n")
		debug.PrintStack()
	} else {
		userDetails = o.GitProvider.UserAuth()
		username = o.GitProvider.CurrentUsername()

		// lets check if we need to fork the repository...
		if originalOrg != username && username != "" && originalOrg != "" && provider.ShouldForkForPullRequest(originalOrg, originalRepo, username) {
			fork = true
		}
	}

	dir := filepath.Join(environmentsDir, gitInfo.Organisation, gitInfo.Name)

	base := source.Ref
	if base == "" {
		base = "master"
	}

	if fork {
		if o.GitProvider == nil {
			return "", "", nil, fork, errors.Wrapf(err, "no Git Provider specified for git URL %s", gitURL)
		}
		repo, err := provider.GetRepository(username, originalRepo)
		if err != nil {
			// lets try create a fork - using a blank organisation to force a user specific fork
			repo, err = provider.ForkRepository(originalOrg, originalRepo, "")
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "failed to fork GitHub repo %s/%s to user %s", originalOrg, originalRepo, username)
			}
			log.Infof("Forked Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
		}

		// lets only use this repository if it is a fork
		if !repo.Fork {
			fork = false
		} else {
			dir, err = ioutil.TempDir("", fmt.Sprintf("fork-%s-%s", gitInfo.Organisation, gitInfo.Name))
			if err != nil {
				return "", "", nil, fork, errors.Wrap(err, "failed to create temp dir")
			}

			err = os.MkdirAll(dir, util.DefaultWritePermissions)
			if err != nil {
				return "", "", nil, fork, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
			}
			cloneGitURL, err := git.CreatePushURL(repo.CloneURL, &userDetails)
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "failed to get clone URL from %s and user %s", repo.CloneURL, username)
			}
			err = o.Gitter.Clone(cloneGitURL, dir)
			if err != nil {
				return "", "", nil, fork, err
			}
			err = git.SetRemoteURL(dir, "upstream", gitURL)
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "setting remote upstream %q in forked environment repo", gitURL)
			}
			if o.ConfigGitFn != nil {
				err = o.ConfigGitFn(dir, gitInfo, o.Gitter)
				if err != nil {
					return "", "", nil, fork, err
				}
			}
			if base != "master" {
				err = o.Gitter.Checkout(dir, base)
				if err != nil {
					return "", "", nil, fork, err
				}
			}
			err = git.ResetToUpstream(dir, base)
			if err != nil {
				return "", "", nil, fork, errors.Wrapf(err, "resetting forked branch %s to upstream version", base)
			}
			return dir, base, gitInfo, fork, nil
		}
	}

	// now lets clone the fork and pull it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return "", "", nil, fork, errors.Wrapf(err, "failed to check if directory %s exists", dir)
	}

	if exists {
		if o.ConfigGitFn != nil {
			err = o.ConfigGitFn(dir, gitInfo, o.Gitter)
			if err != nil {
				return "", "", nil, fork, err
			}
		}
		// lets check the git remote URL is setup correctly
		err = o.Gitter.SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return "", "", nil, fork, err
		}
		err = o.Gitter.Stash(dir)
		if err != nil {
			return "", "", nil, fork, err
		}
		err = o.Gitter.Checkout(dir, base)
		if err != nil {
			return "", "", nil, fork, err
		}
		err = o.Gitter.Pull(dir)
		if err != nil {
			return "", "", nil, fork, err
		}
	} else {
		err := os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return "", "", nil, fork, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		cloneGitURL, err := git.CreatePushURL(gitURL, &userDetails)
		if err != nil {
			return "", "", nil, fork, errors.Wrapf(err, "failed to get clone URL from %s and user %s", gitURL, username)
		}

		err = o.Gitter.Clone(cloneGitURL, dir)
		if err != nil {
			return "", "", nil, fork, err
		}
		if o.ConfigGitFn != nil {
			err = o.ConfigGitFn(dir, gitInfo, o.Gitter)
			if err != nil {
				return "", "", nil, fork, err
			}
		}
		if base != "master" {
			err = o.Gitter.Checkout(dir, base)
			if err != nil {
				return "", "", nil, fork, err
			}
		}
	}
	return dir, base, gitInfo, fork, nil
}

// CreateUpgradeRequirementsFn creates the ModifyChartFn that upgrades the requirements of a chart.
// Either all requirements may be upgraded, or the chartName,
// alias and version can be specified. A username and password can be passed for a protected repository.
// The passed inspectChartFunc will be called whilst the chart for each requirement is unpacked on the disk.
// Operations are carried out using the helmer interface and there will be more logging if verbose is true.
// The passed valuesFiles are used to add a values.yaml to each requirement.
func CreateUpgradeRequirementsFn(all bool, chartName string, alias string, version string, username string,
	password string, helmer helm.Helmer, inspectChartFunc func(chartDir string,
		existingValues map[string]interface{}) error, verbose bool, valuesFiles *ValuesFiles) ModifyChartFn {
	upgraded := false
	return func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
		templates map[string]string, envDir string, details *PullRequestDetails) error {

		// Work through the upgrades
		for _, d := range requirements.Dependencies {
			// We need to ignore the platform unless the chart name is the platform
			upgrade := false
			if all {
				if d.Name != "jenkins-x-platform" {
					upgrade = true
				}
			} else {
				if d.Name == chartName && (d.Alias == "" || d.Alias == alias) {
					upgrade = true
				}
			}
			if upgrade {
				upgraded = true

				oldVersion := d.Version
				err := helm.InspectChart(d.Name, version, d.Repository, username, password, helmer,
					func(chartDir string) error {
						if all || version == "" {
							// Upgrade to the latest version
							_, chartVersion, err := helm.LoadChartNameAndVersion(filepath.Join(chartDir, "Chart.yaml"))
							if err != nil {
								return errors.Wrapf(err, "error loading chart from %s", chartDir)
							}
							version = chartVersion
							if verbose {
								log.Infof("No version specified so using latest version which is %s\n", util.ColorInfo(version))
							}
						}

						err := inspectChartFunc(chartDir, values)
						if err != nil {
							return errors.Wrapf(err, "running inspectChartFunc for %s", d.Name)
						}
						err = CreateNestedRequirementDir(envDir, chartName, chartDir, version, d.Repository, verbose,
							valuesFiles, helmer)
						if err != nil {
							return errors.Wrapf(err, "creating nested app dir in chart dir %s", chartDir)
						}
						return nil
					})
				if err != nil {
					return errors.Wrapf(err, "inspecting chart %s", d.Name)
				}

				// Do the upgrade
				d.Version = version
				if !all {
					details.Title = fmt.Sprintf("Upgrade %s to %s", chartName, version)
					details.Message = fmt.Sprintf("Upgrade %s from %s to %s", chartName, oldVersion, version)
				} else {
					details.Message = fmt.Sprintf("%s\n* %s from %s to %s", details.Message, d.Name, oldVersion, version)
				}
			}
		}
		if !upgraded {
			log.Infof("No upgrades available\n")
		}
		return nil
	}
}

// CreateAddRequirementFn create the ModifyChartFn that adds a dependency to a chart. It takes the chart name,
// an alias for the chart, the version of the chart, the repo to load the chart from,
// valuesFiles (an array of paths to values.yaml files to add). The chartDir is the unpacked chart being added,
// which is used to add extra metadata about the chart (e.g. the charts readme, the release.yaml, the git repo url and
// the release notes) - if this points to a non-existent directory it will be ignored.
func CreateAddRequirementFn(chartName string, alias string, version string, repo string,
	valuesFiles *ValuesFiles, chartDir string, verbose bool, helmer helm.Helmer) ModifyChartFn {
	return func(requirements *helm.Requirements, chart *helmchart.Metadata, values map[string]interface{},
		templates map[string]string, envDir string, details *PullRequestDetails) error {
		// See if the chart already exists in requirements
		found := false
		for _, d := range requirements.Dependencies {
			if d.Name == chartName && d.Alias == alias {
				// App found
				log.Infof("App %s already installed.\n", util.ColorWarning(chartName))
				if version != d.Version {
					log.Infof("To upgrade the chartName use %s or %s\n",
						util.ColorInfo("jx upgrade chartName <chartName>"),
						util.ColorInfo("jx upgrade apps --all"))
				}
				found = true
				break
			}
		}
		// If chartName not found, add it
		if !found {
			requirements.Dependencies = append(requirements.Dependencies, &helm.Dependency{
				Alias:      alias,
				Repository: repo,
				Name:       chartName,
				Version:    version,
			})
			err := CreateNestedRequirementDir(envDir, chartName, chartDir, version, repo, verbose, valuesFiles, helmer)
			if err != nil {
				return errors.Wrapf(err, "creating nested app dir in chart dir %s", chartDir)
			}

		}
		return nil
	}
}

// CreateNestedRequirementDir creates the a directory for a chart being added as a requirement, adding a README.md,
// the release.yaml, and the values.yaml. The dir is the unpacked chart directory to which the requirement is being
// added. The requirementName, requirementVersion,
// requirementRepository and requirementValuesFiles are used to construct the metadata,
// as well as info in the requirementDir which points to the unpacked chart of the requirement.
func CreateNestedRequirementDir(dir string, requirementName string, requirementDir string, requirementVersion string,
	requirementRepository string, verbose bool, requirementValuesFiles *ValuesFiles, helmer helm.Helmer) error {
	appDir := filepath.Join(dir, requirementName)
	rootValuesFileName := filepath.Join(appDir, helm.ValuesFileName)
	err := os.MkdirAll(appDir, 0700)
	if err != nil {
		return errors.Wrapf(err, "cannot create requirementName directory %s", appDir)
	}
	if verbose {
		log.Infof("Using %s for requirementName files\n", appDir)
	}
	if requirementValuesFiles != nil && len(requirementValuesFiles.Items) == 1 {
		// We need to write the values file into the right spot for the requirementName
		err = util.CopyFile(requirementValuesFiles.Items[0], rootValuesFileName)
		if err != nil {
			return errors.Wrapf(err, "cannot copy values."+
				"yaml to %s directory %s", requirementName, appDir)
		}
		if verbose {
			log.Infof("Writing values file to %s\n", rootValuesFileName)
		}
	}
	// Write the release.yaml
	var gitRepo, releaseNotesURL, appReadme, description string
	templatesDir := filepath.Join(requirementDir, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		if verbose {
			log.Infof("No templates directory exists in %s\n", util.ColorInfo(dir))
		}
	} else if err != nil {
		return errors.Wrapf(err, "stat directory %s", appDir)
	} else {
		releaseYamlPath := filepath.Join(templatesDir, "release.yaml")
		if _, err := os.Stat(releaseYamlPath); err == nil {
			bytes, err := ioutil.ReadFile(releaseYamlPath)
			if err != nil {
				return errors.Wrapf(err, "release.yaml from %s", templatesDir)
			}
			release := jenkinsv1.Release{}
			err = yaml.Unmarshal(bytes, &release)
			if err != nil {
				return errors.Wrapf(err, "unmarshal %s", releaseYamlPath)
			}
			gitRepo = release.Spec.GitHTTPURL
			releaseNotesURL = release.Spec.ReleaseNotesURL
			releaseYamlOutPath := filepath.Join(appDir, "release.yaml")
			err = ioutil.WriteFile(releaseYamlOutPath, bytes, 0755)
			if err != nil {
				return errors.Wrapf(err, "write file %s", releaseYamlOutPath)
			}
			if verbose {
				log.Infof("Read release notes URL %s and git repo url %s from release.yaml\nWriting release."+
					"yaml from chartName to %s\n", releaseNotesURL, gitRepo, releaseYamlOutPath)
			}
		} else if os.IsNotExist(err) {
			if verbose {

				log.Infof("Not adding release.yaml as not present in chart. Only files in %s are:\n",
					templatesDir)
				err := util.ListDirectory(templatesDir, true)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.Wrapf(err, "reading release.yaml from %s", templatesDir)
		}
	}
	chartYamlPath := filepath.Join(requirementDir, helm.ChartFileName)
	if _, err := os.Stat(chartYamlPath); err == nil {
		bytes, err := ioutil.ReadFile(chartYamlPath)
		if err != nil {
			return errors.Wrapf(err, "read %s from %s", helm.ChartFileName, requirementDir)
		}
		chart := helmchart.Metadata{}
		err = yaml.Unmarshal(bytes, &chart)
		if err != nil {
			return errors.Wrapf(err, "unmarshal %s", chartYamlPath)
		}
		description = chart.Description
	} else if os.IsNotExist(err) {
		if verbose {
			log.Infof("Not adding %s as not present in chart. Only files in %s are:\n", helm.ChartFileName,
				requirementDir)
			err := util.ListDirectory(requirementDir, true)
			if err != nil {
				return err
			}
		}
	} else {
		return errors.Wrapf(err, "stat Chart.yaml from %s", requirementDir)
	}
	// Need to copy over any referenced files, and their schemas
	rootValues, err := helm.LoadValuesFile(rootValuesFileName)
	if err != nil {
		return err
	}
	schemas := make(map[string][]string)
	possibles := make(map[string]string)
	if _, err := os.Stat(requirementDir); err == nil {
		files, err := ioutil.ReadDir(requirementDir)
		if err != nil {
			return errors.Wrapf(err, "unable to list files in %s", requirementDir)
		}
		possibleReadmes := make([]string, 0)
		for _, file := range files {
			fileName := strings.ToUpper(file.Name())
			if fileName == "README.MD" || fileName == "README" {
				possibleReadmes = append(possibleReadmes, filepath.Join(requirementDir, file.Name()))
			}
		}
		if len(possibleReadmes) > 1 {
			if verbose {
				log.Warnf("Unable to add README to PR for %s as more than one exists and not sure which to"+
					" use %s\n", requirementName, possibleReadmes)
			}
		} else if len(possibleReadmes) == 1 {
			bytes, err := ioutil.ReadFile(possibleReadmes[0])
			if err != nil {
				return errors.Wrapf(err, "unable to read file %s", possibleReadmes[0])
			}
			appReadme = string(bytes)
		}

		for _, f := range files {
			ignore, err := util.IgnoreFile(f.Name(), helm.DefaultValuesTreeIgnores)
			if err != nil {
				return err
			}
			if !f.IsDir() && !ignore {
				key := f.Name()
				// Handle .schema. files specially
				if parts := strings.Split(key, ".schema."); len(parts) > 1 {
					// this is a file named *.schema.*, the part before .schema is the key
					if _, ok := schemas[parts[0]]; !ok {
						schemas[parts[0]] = make([]string, 0)
					}
					schemas[parts[0]] = append(schemas[parts[0]], filepath.Join(requirementDir, f.Name()))
				}
				possibles[key] = filepath.Join(requirementDir, f.Name())

			}
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, fmt.Sprintf("error reading %s", requirementDir))
	}
	if verbose && appReadme == "" {
		log.Infof("Not adding App Readme as no README, README.md, readme or readme.md found in %s\n", requirementDir)
	}
	UpdateAppResource(helmer, requirementDir, appDir, requirementName, requirementRepository)
	if err != nil {
		return err
	}
	readme := helm.GenerateReadmeForChart(requirementName, requirementVersion, description, requirementRepository, gitRepo, releaseNotesURL, appReadme)
	readmeOutPath := filepath.Join(appDir, "README.MD")
	err = ioutil.WriteFile(readmeOutPath, []byte(readme), 0755)
	if err != nil {
		return errors.Wrapf(err, "write README.md to %s", appDir)

		if verbose {
			log.Infof("Writing README.md to %s\n", readmeOutPath)
		}
		externalFileHandler := func(path string, element map[string]interface{}, key string) error {
			fileName, _ := filepath.Split(path)
			err := util.CopyFile(path, filepath.Join(appDir, fileName))
			if err != nil {
				return errors.Wrapf(err, "copy %s to %s", path, appDir)
			}
			// key for schema is the filename without the extension
			schemaKey := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			if schemaPaths, ok := schemas[schemaKey]; ok {
				for _, schemaPath := range schemaPaths {
					fileName, _ := filepath.Split(schemaPath)
					schemaOutPath := filepath.Join(appDir, fileName)
					err := util.CopyFile(schemaPath, schemaOutPath)
					if err != nil {
						return errors.Wrapf(err, "copy %s to %s", schemaPath, appDir)
					}
					if verbose {
						log.Infof("Writing %s to %s\n", fileName, schemaOutPath)
					}
				}
			}
			return nil
		}
		err = helm.HandleExternalFileRefs(rootValues, possibles, "", externalFileHandler)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddAppMetaData applies chart metadata to an App resource
func AddAppMetaData(chartDir string, app *jenkinsv1.App, repository string) (*jenkinsv1.App, error) {
	metadata, err := helm.LoadChartFile(filepath.Join(chartDir, "Chart.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, "error loading chart from %s", chartDir)
	}
	if app.Annotations == nil {
		app.Annotations = make(map[string]string)
	}
	app.Annotations[helm.AnnotationAppDescription] = metadata.GetDescription()
	repoURL, err := url.Parse(repository)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid repository url")
	}
	app.Annotations[helm.AnnotationAppRepository] = util.StripCredentialsFromURL(repoURL)
	if app.Labels == nil {
		app.Labels = make(map[string]string)
	}
	app.Labels[helm.LabelAppName] = metadata.Name
	app.Labels[helm.LabelAppVersion] = metadata.Version
	return app, nil
}

// UpdateAppResource updates app resource template with app specific metadata
func UpdateAppResource(helmer helm.Helmer, fetchedChartDir string, outputDir string, name string, repository string) error {
	outputTemplateDir := filepath.Join(outputDir, "templates")
	templatesDirExists, err := util.DirExists(outputTemplateDir)
	if err != nil {
		return err
	}
	if !templatesDirExists {
		os.Mkdir(outputTemplateDir, os.ModePerm)
	}

	templateWorkDir := filepath.Join(fetchedChartDir, "output")
	templateWorkDirExists, err := util.DirExists(templateWorkDir)
	if err != nil {
		return err
	}
	if !templateWorkDirExists {
		os.Mkdir(templateWorkDir, os.ModePerm)
	}
	err = helmer.Template(fetchedChartDir, name, "", templateWorkDir, false, make([]string, 0), make([]string, 0))
	if err != nil {
		return err
	}
	completedTemplatesDir := filepath.Join(templateWorkDir, name, "templates")
	templates, _ := ioutil.ReadDir(completedTemplatesDir)
	for _, template := range templates {
		app := &jenkinsv1.App{}
		appBytes, err := ioutil.ReadFile(filepath.Join(completedTemplatesDir, template.Name()))
		if err == nil {
			err = yaml.Unmarshal(appBytes, app)
			if err == nil {
				if app.Kind == "App" {
					// Enhance the first app resource found
					AddAppMetaData(fetchedChartDir, app, repository)
					outputTemplateFile := filepath.Join(outputTemplateDir, template.Name())
					helm.SaveFile(outputTemplateFile, app)
					return nil
				}
			}
		}
	}
	outputTemplateFile := filepath.Join(outputTemplateDir, name+"-app.yaml")
	// No app resource was found so auto generate one
	app := &jenkinsv1.App{
		TypeMeta: metav1.TypeMeta{
			Kind:       "App",
			APIVersion: jenkinsio.GroupName + "/" + jenkinsio.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: jenkinsv1.AppSpec{},
	}
	AddAppMetaData(fetchedChartDir, app, repository)
	err = helm.SaveFile(outputTemplateFile, app)
	return err
}
