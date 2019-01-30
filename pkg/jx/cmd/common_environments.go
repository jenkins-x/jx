package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/ghodss/yaml"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
)

// ModifyChartFn callback for modifying a chart, requirements, the chart metadata,
// the values.yaml and all files in templates are unmarshaled, and the root dir for the chart is passed
type ModifyChartFn func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
	templates map[string]string, dir string) error

// ConfigureGitFolderFn callback to optionally configure git before its used for creating commits and PRs
type ConfigureGitFolderFn func(dir string, gitInfo *gits.GitRepository, gitAdapter gits.Gitter) error

// CreateEnvPullRequestFn callback that allows the pull request creation to be mocked out
type CreateEnvPullRequestFn func(env *jenkinsv1.Environment, modifyChartFn ModifyChartFn, branchNameText string,
	title string, message string, pullRequestInfo *gits.PullRequestInfo) (*gits.PullRequestInfo, error)

func (o *CommonOptions) createEnvironmentPullRequest(env *jenkinsv1.Environment, modifyChartFn ModifyChartFn,
	branchNameText *string, title *string, message *string, pullRequestInfo *gits.PullRequestInfo,
	configGitFn ConfigureGitFolderFn) (*gits.PullRequestInfo, error) {
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

	environmentsDir, err := o.EnvironmentsDir()
	if err != nil {
		return answer, err
	}
	dir := filepath.Join(environmentsDir, gitInfo.Organisation, gitInfo.Name)

	// now lets clone the fork and push it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return answer, err
	}

	branchName := o.Git().ConvertToValidBranchName(asText(branchNameText))
	base := source.Ref
	if base == "" {
		base = "master"
	}

	if exists {
		if configGitFn != nil {
			err = configGitFn(dir, gitInfo, o.Git())
			if err != nil {
				return answer, err
			}
		}
		// lets check the git remote URL is setup correctly
		err = o.Git().SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return answer, err
		}
		err = o.Git().Stash(dir)
		if err != nil {
			return answer, err
		}
		err = o.Git().Checkout(dir, base)
		if err != nil {
			return answer, err
		}
		err = o.Git().Pull(dir)
		if err != nil {
			return answer, err
		}
	} else {
		err := os.MkdirAll(dir, DefaultWritePermissions)
		if err != nil {
			return answer, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		err = o.Git().Clone(gitURL, dir)
		if err != nil {
			return answer, err
		}
		if configGitFn != nil {
			err = configGitFn(dir, gitInfo, o.Git())
			if err != nil {
				return answer, err
			}
		}
		if base != "master" {
			err = o.Git().Checkout(dir, base)
			if err != nil {
				return answer, err
			}
		}

		// TODO lets fork if required???
	}
	branchNames, err := o.Git().RemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return answer, fmt.Errorf("Failed to load remote branch names: %s", err)
	}
	//log.Infof("Found remote branch names %s\n", strings.Join(branchNames, ", "))
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchName += "-" + string(uuid.NewUUID())
	}
	err = o.Git().CreateBranch(dir, branchName)
	if err != nil {
		return answer, err
	}
	err = o.Git().Checkout(dir, branchName)
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

	err = modifyChartFn(requirements, chart, values, templates, dir)
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

	err = o.Git().Add(dir, "*", "*/*")
	if err != nil {
		return answer, err
	}
	changed, err := o.Git().HasChanges(dir)
	if err != nil {
		return answer, err
	}
	if !changed {
		log.Warnf("%s\n", "No changes made to the GitOps Environment source code. Code must be up to date!")
		return answer, nil
	}
	err = o.Git().CommitDir(dir, asText(message))
	if err != nil {
		return answer, err
	}
	// lets rebase an existing PR
	if pullRequestInfo != nil {
		remoteBranch := pullRequestInfo.PullRequestArguments.Head
		err = o.Git().ForcePushBranch(dir, branchName, remoteBranch)
		return pullRequestInfo, err
	}

	err = o.Git().Push(dir)
	if err != nil {
		return answer, err
	}

	provider, err := o.gitProviderForURL(gitURL, "user name to submit the Pull Request")
	if err != nil {
		return answer, err
	}

	gha := &gits.GitPullRequestArguments{
		GitRepository: gitInfo,
		Title:         asText(title),
		Body:          asText(message),
		Base:          base,
		Head:          branchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
		return answer, err
	}
	log.Infof("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return &gits.PullRequestInfo{
		GitProvider:          provider,
		PullRequest:          pr,
		PullRequestArguments: gha,
	}, nil
}

func (o *CommonOptions) registerEnvironmentCRD() error {
	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterEnvironmentCRD(apisClient)
	return err
}

// modifyDevEnvironment performs some mutation on the Development environemnt to modify team settings
func (o *CommonOptions) modifyDevEnvironment(jxClient versioned.Interface, ns string,
	fn func(env *jenkinsv1.Environment) error) error {
	env, err := kube.EnsureDevEnvironmentSetup(jxClient, ns)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure that dev environment is setup for namespace '%s'", ns)
	}
	if env == nil {
		return fmt.Errorf("No Development environment found in namespace %s", ns)
	}
	err = fn(env)
	if err != nil {
		return errors.Wrap(err, "failed to call the callback function for dev environment")
	}
	_, err = jxClient.JenkinsV1().Environments(ns).Update(env)
	if err != nil {
		return fmt.Errorf("Failed to update Development environment in namespace %s: %s", ns, err)
	}
	return nil
}

func asText(text *string) string {
	if text != nil {
		return *text
	}
	return ""
}

// CreateAddRequirementFn create the ModifyChartFn that adds a dependency to a chart. It takes the chart name,
// an alias for the chart, the version of the chart, the repo to load the chart from,
// valuesFiles (an array of paths to values.yaml files to add). The chartDir is the unpacked chart being added,
// which is used to add extra metadata about the chart (e.g. the charts readme, the release.yaml, the git repo url and
// the release notes) - if this points to a non-existant directory it will be ignored.
func (o *CommonOptions) CreateAddRequirementFn(chartName string, alias string, version string, repo string,
	valuesFiles []string, chartDir string) ModifyChartFn {
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
			if o.Verbose {
				log.Infof("Using %s for chartName files\n", appDir)
			}
			if len(valuesFiles) == 1 {
				// We need to write the values file into the right spot for the chartName
				err = util.CopyFile(valuesFiles[0], rootValuesFileName)
				if err != nil {
					return errors.Wrapf(err, "cannot copy values."+
						"yaml to chartName directory %s", appDir)
				}
				if o.Verbose {
					log.Infof("Writing values file to %s\n", rootValuesFileName)
				}
			}
			// Write the release.yaml
			var gitRepo, releaseNotesURL, appReadme, description string
			templatesDir := filepath.Join(chartDir, "templates")
			if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
				if o.Verbose {
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
					if o.Verbose {
						log.Infof("Read release notes URL %s and git repo url %s from release.yaml\nWriting release."+
							"yaml from chartName to %s\n", releaseNotesURL, gitRepo, releaseYamlOutPath)
					}
				} else if os.IsNotExist(err) {
					if o.Verbose {

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
				if o.Verbose {
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
					if o.Verbose {
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
			if o.Verbose && appReadme == "" {
				log.Infof("Not adding App Readme as no README, README.md, readme or readme.md found in %s\n", chartDir)
			}
			readme := helm.GenerateReadmeForChart(chartName, version, description, repo, gitRepo, releaseNotesURL, appReadme)
			readmeOutPath := filepath.Join(appDir, "README.MD")
			err = ioutil.WriteFile(readmeOutPath, []byte(readme), 0755)
			if err != nil {
				return errors.Wrapf(err, "write README.md to %s", appDir)

				if o.Verbose {
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
							if o.Verbose {
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
