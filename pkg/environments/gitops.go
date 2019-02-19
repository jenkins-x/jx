package environments

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"k8s.io/helm/pkg/proto/hapi/chart"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
)

// ModifyChartFn callback for modifying a chart, requirements, the chart metadata,
// the values.yaml and all files in templates are unmarshaled, and the root dir for the chart is passed
type ModifyChartFn func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
	templates map[string]string, dir string) error

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
func (o *EnvironmentPullRequestOptions) Create(env *jenkinsv1.Environment, branchNameText *string, title *string,
	message *string, environmentsDir string, pullRequestInfo *gits.PullRequestInfo) (*gits.PullRequestInfo, error) {
	var answer *gits.PullRequestInfo
	source := &env.Spec.Source
	gitURL := source.URL
	if gitURL == "" {
		return answer, fmt.Errorf("No source git URL")
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return answer, err
	}

	if err != nil {
		return answer, err
	}
	dir := filepath.Join(environmentsDir, gitInfo.Organisation, gitInfo.Name)

	// now lets clone the fork and push it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return answer, err
	}

	branchName := o.Gitter.ConvertToValidBranchName(util.DereferenceString(branchNameText))
	base := source.Ref
	if base == "" {
		base = "master"
	}

	if exists {
		if o.ConfigGitFn != nil {
			err = o.ConfigGitFn(dir, gitInfo, o.Gitter)
			if err != nil {
				return answer, err
			}
		}
		// lets check the git remote URL is setup correctly
		err = o.Gitter.SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return answer, err
		}
		err = o.Gitter.Stash(dir)
		if err != nil {
			return answer, err
		}
		err = o.Gitter.Checkout(dir, base)
		if err != nil {
			return answer, err
		}
		err = o.Gitter.Pull(dir)
		if err != nil {
			return answer, err
		}
	} else {
		err := os.MkdirAll(dir, util.DefaultWritePermissions)
		if err != nil {
			return answer, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		err = o.Gitter.Clone(gitURL, dir)
		if err != nil {
			return answer, err
		}
		if o.ConfigGitFn != nil {
			err = o.ConfigGitFn(dir, gitInfo, o.Gitter)
			if err != nil {
				return answer, err
			}
		}
		if base != "master" {
			err = o.Gitter.Checkout(dir, base)
			if err != nil {
				return answer, err
			}
		}

		// TODO lets fork if required???
	}
	branchNames, err := o.Gitter.RemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return answer, errors.Wrapf(err, "Failed to load remote branch names")
	}
	//log.Infof("Found remote branch names %s\n", strings.Join(branchNames, ", "))
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchName += "-" + string(uuid.NewV4().String())
	}
	err = o.Gitter.CreateBranch(dir, branchName)
	if err != nil {
		return answer, err
	}
	err = o.Gitter.Checkout(dir, branchName)
	if err != nil {
		return answer, err
	}

	requirementsFile, err := helm.FindRequirementsFileName(dir)
	if err != nil {
		return answer, err
	}
	requirements, err := helm.LoadRequirementsFile(requirementsFile)
	if err != nil {
		return answer, err
	}

	chartFile, err := helm.FindChartFileName(dir)
	if err != nil {
		return answer, err
	}
	chart, err := helm.LoadChartFile(chartFile)
	if err != nil {
		return answer, err
	}

	valuesFile, err := helm.FindValuesFileName(dir)
	if err != nil {
		return answer, err
	}
	values, err := helm.LoadValuesFile(valuesFile)
	if err != nil {
		return answer, err
	}

	templatesDir, err := helm.FindTemplatesDirName(dir)
	if err != nil {
		return answer, err
	}
	templates, err := helm.LoadTemplatesDir(templatesDir)
	if err != nil {
		return answer, err
	}

	err = o.ModifyChartFn(requirements, chart, values, templates, dir)
	if err != nil {
		return answer, err
	}

	err = helm.SaveFile(requirementsFile, requirements)
	if err != nil {
		return answer, err
	}

	err = helm.SaveFile(chartFile, chart)
	if err != nil {
		return answer, err
	}

	err = helm.SaveFile(valuesFile, values)
	if err != nil {
		return answer, err
	}

	err = o.Gitter.Add(dir, "-A")
	if err != nil {
		return answer, err
	}
	changed, err := o.Gitter.HasChanges(dir)
	if err != nil {
		return answer, err
	}
	if !changed {
		log.Warnf("%s\n", "No changes made to the GitOps Environment source code. Code must be up to date!")
		return answer, nil
	}
	err = o.Gitter.CommitDir(dir, util.DereferenceString(message))
	if err != nil {
		return answer, err
	}
	// lets rebase an existing PR
	if pullRequestInfo != nil {
		remoteBranch := pullRequestInfo.PullRequestArguments.Head
		err = o.Gitter.ForcePushBranch(dir, branchName, remoteBranch)
		return pullRequestInfo, err
	}

	err = o.Gitter.Push(dir)
	if err != nil {
		return answer, err
	}

	gha := &gits.GitPullRequestArguments{
		GitRepository: gitInfo,
		Title:         util.DereferenceString(title),
		Body:          util.DereferenceString(message),
		Base:          base,
		Head:          branchName,
	}

	pr, err := o.GitProvider.CreatePullRequest(gha)
	if err != nil {
		return answer, err
	}
	log.Infof("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return &gits.PullRequestInfo{
		GitProvider:          o.GitProvider,
		PullRequest:          pr,
		PullRequestArguments: gha,
	}, nil
}

// CreateAddRequirementFn create the ModifyChartFn that adds a dependency to a chart. It takes the chart name,
// an alias for the chart, the version of the chart, the repo to load the chart from,
// valuesFiles (an array of paths to values.yaml files to add). The chartDir is the unpacked chart being added,
// which is used to add extra metadata about the chart (e.g. the charts readme, the release.yaml, the git repo url and
// the release notes) - if this points to a non-existant directory it will be ignored.
func CreateAddRequirementFn(chartName string, alias string, version string, repo string,
	valuesFiles []string, chartDir string, verbose bool) ModifyChartFn {
	return func(requirements *helm.Requirements, chart *helmchart.Metadata, values map[string]interface{},
		templates map[string]string, envDir string) error {
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
			appDir := filepath.Join(envDir, chartName)
			rootValuesFileName := filepath.Join(appDir, helm.ValuesFileName)
			err := os.MkdirAll(appDir, 0700)
			if err != nil {
				return errors.Wrapf(err, "cannot create chartName directory %s", appDir)
			}
			if verbose {
				log.Infof("Using %s for chartName files\n", appDir)
			}
			if len(valuesFiles) == 1 {
				// We need to write the values file into the right spot for the chartName
				err = util.CopyFile(valuesFiles[0], rootValuesFileName)
				if err != nil {
					return errors.Wrapf(err, "cannot copy values."+
						"yaml to %s directory %s", chartName, appDir)
				}
				if verbose {
					log.Infof("Writing values file to %s\n", rootValuesFileName)
				}
			}
			// Write the release.yaml
			var gitRepo, releaseNotesURL, appReadme, description string
			templatesDir := filepath.Join(chartDir, "templates")
			if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
				if verbose {
					log.Infof("No templates directory exists in %s", chartDir)
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
			chartYamlPath := filepath.Join(chartDir, helm.ChartFileName)
			if _, err := os.Stat(chartYamlPath); err == nil {
				bytes, err := ioutil.ReadFile(chartYamlPath)
				if err != nil {
					return errors.Wrapf(err, "read %s from %s", helm.ChartFileName, chartDir)
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
						chartDir)
					err := util.ListDirectory(chartDir, true)
					if err != nil {
						return err
					}
				}
			} else {
				return errors.Wrapf(err, "stat Chart.yaml from %s", chartDir)
			}
			// Need to copy over any referenced files, and their schemas
			rootValues, err := helm.LoadValuesFile(rootValuesFileName)
			if err != nil {
				return err
			}
			schemas := make(map[string][]string)
			possibles := make(map[string]string)
			if _, err := os.Stat(chartDir); err == nil {
				files, err := ioutil.ReadDir(chartDir)
				if err != nil {
					return errors.Wrapf(err, "unable to list files in %s", chartDir)
				}
				possibleReadmes := make([]string, 0)
				for _, file := range files {
					fileName := strings.ToUpper(file.Name())
					if fileName == "README.MD" || fileName == "README" {
						possibleReadmes = append(possibleReadmes, filepath.Join(chartDir, file.Name()))
					}
				}
				if len(possibleReadmes) > 1 {
					if verbose {
						log.Warnf("Unable to add README to PR for %s as more than one exists and not sure which to"+
							" use %s\n", chartName, possibleReadmes)
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
							schemas[parts[0]] = append(schemas[parts[0]], filepath.Join(chartDir, f.Name()))
						}
						possibles[key] = filepath.Join(chartDir, f.Name())

					}
				}
			} else if !os.IsNotExist(err) {
				return errors.Wrap(err, fmt.Sprintf("error reading %s", chartDir))
			}
			if verbose && appReadme == "" {
				log.Infof("Not adding App Readme as no README, README.md, readme or readme.md found in %s\n", chartDir)
			}
			readme := helm.GenerateReadmeForChart(chartName, version, description, repo, gitRepo, releaseNotesURL, appReadme)
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
		}
		return nil
	}
}
