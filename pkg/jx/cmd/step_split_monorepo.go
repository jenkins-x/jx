package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	optionOrganisation   = "organisation"
	defaultKubernetesDir = "kubernetes"
)

var (
	stepSplitMonorepoOptions = []string{optionMinJxVersion}

	stepSplitMonorepoLong = templates.LongDesc(`
		Mirrors the code from a monorepo into separate microservice style Git repositories so its easier to do finer grained releases.

		If you have lots of apps in folders in a monorepo then this command can run on that repo to mirror changes into a number of microservice based repositories which can each then get auto-imported into Jenkins X

`)

	stepSplitMonorepoExample = templates.Examples(`
		# Split the current folder up into separate Git repositories 
		jx step split monorepo -o mygithuborg
			`)
)

// StepSplitMonorepoOptions contains the command line flags
type StepSplitMonorepoOptions struct {
	StepOptions

	Glob          string
	Organisation  string
	Dir           string
	OutputDir     string
	KubernetesDir string
	NoGit         bool
}

// NewCmdStepSplitMonorepo Creates a new Command object
func NewCmdStepSplitMonorepo(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepSplitMonorepoOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "split monorepo",
		Short:   "Mirrors the code from a monorepo into separate microservice style Git repositories so its easier to do finer grained releases",
		Long:    stepSplitMonorepoLong,
		Example: stepSplitMonorepoExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Glob, "glob", "g", "*", "The glob pattern to find folders to mirror to separate repositories")
	cmd.Flags().StringVarP(&options.Organisation, optionOrganisation, "o", "", "The GitHub organisation to split the repositories into")
	cmd.Flags().StringVarP(&options.Dir, "source-dir", "s", "", "The source directory to look inside for the folders to move into separate Git repositories")
	cmd.Flags().StringVarP(&options.OutputDir, optionOutputDir, "d", "generated", "The output directory where new projects are created")
	cmd.Flags().StringVarP(&options.KubernetesDir, "kubernetes-folder", "", defaultKubernetesDir, "The folder containing all the Kubernetes YAML for each app")
	cmd.Flags().BoolVarP(&options.NoGit, "no-git", "", false, "If enabled then don't try to clone/create the separate repositories in github")
	return cmd
}

// Run implements this command
func (o *StepSplitMonorepoOptions) Run() error {
	organisation := o.Organisation
	if organisation == "" {
		return util.MissingOption(optionOrganisation)
	}
	outputDir := o.OutputDir
	if outputDir == "" {
		return util.MissingOption(optionOutputDir)
	}
	var err error
	dir := o.Dir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	glob := o.Glob

	fullGlob := filepath.Join(dir, glob)
	o.Debugf("Searching in monorepo at: %s\n", fullGlob)
	matches, err := filepath.Glob(fullGlob)
	if err != nil {
		return err
	}
	kubeDir := o.KubernetesDir
	if kubeDir == "" {
		kubeDir = defaultKubernetesDir
	}
	var gitProvider gits.GitProvider
	if !o.NoGit {
		gitProvider, err = o.createGitProviderForURL(gits.KindGitHub, gits.GitHubURL)
		if err != nil {
			return err
		}
	}

	for _, path := range matches {
		_, name := filepath.Split(path)
		if !strings.HasPrefix(name, ".") && name != kubeDir {
			fi, err := os.Stat(path)
			if err != nil {
				return err
			}
			switch mode := fi.Mode(); {
			case mode.IsDir():
				o.Debugf("Found match: %s\n", path)
				outPath := filepath.Join(outputDir, name)

				var gitUrl string
				var repo *gits.GitRepository
				createRepo := true
				if !o.NoGit {
					// lets clone the project if it exists
					repo, err = gitProvider.GetRepository(organisation, name)
					if repo != nil && err == nil {
						err = os.MkdirAll(outPath, DefaultWritePermissions)
						if err != nil {
							return err
						}
						createRepo = false
						userAuth := gitProvider.UserAuth()
						gitUrl, err = o.Git().CreatePushURL(repo.CloneURL, &userAuth)
						if err != nil {
							return err
						}
						log.Infof("Cloning %s into directory %s\n", util.ColorInfo(repo.CloneURL), util.ColorInfo(outPath))
						err = o.Git().CloneOrPull(gitUrl, outPath)
						if err != nil {
							return err
						}
					}
				}

				err = util.CopyDirOverwrite(path, outPath)
				if err != nil {
					return err
				}

				// lets copy the .gitignore
				localGitIgnore := filepath.Join(outPath, ".gitignore")
				exists, err := util.FileExists(localGitIgnore)
				if err != nil {
					return err
				}
				if !exists {
					rootGitIgnore := filepath.Join(dir, ".gitignore")
					exists, err = util.FileExists(rootGitIgnore)
					if err != nil {
						return err
					}
					if exists {
						err = util.CopyFile(rootGitIgnore, localGitIgnore)
						if err != nil {
							return err
						}
					}
				}

				if !o.NoGit {
					if createRepo {
						repo, err = gitProvider.CreateRepository(organisation, name, false)
						if err != nil {
							return err
						}
						log.Infof("Created Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))

						userAuth := gitProvider.UserAuth()
						gitUrl, err = o.Git().CreatePushURL(repo.CloneURL, &userAuth)

						err := o.Git().Init(outPath)
						if err != nil {
							return err
						}
						err = o.Git().AddRemote(outPath, "origin", gitUrl)
						if err != nil {
							return err
						}
					}
					// ignore errors as probably already added
					o.Git().Add(outPath, ".gitignore")
					o.Git().Add(outPath, "src", "charts", "*")

					message := "generated by: jx step split monorepo"

					err = o.Git().CommitIfChanges(outPath, message)
					if err != nil {
						return err
					}
					err = o.Git().PushMaster(outPath)
					if err != nil {
						return err
					}
					log.Infof("Pushed Git repository to %s\n\n", util.ColorInfo(repo.HTMLURL))
				}
			}
		}
	}
	if kubeDir != "" {

		// now lets copy any Kubernetes YAML into Helm charts in the apps
		matches, err = filepath.Glob(filepath.Join(dir, kubeDir, "*"))
		if err != nil {
			return err
		}
		for _, path := range matches {
			_, name := filepath.Split(path)
			if strings.HasSuffix(name, ".yaml") {
				appName := strings.TrimSuffix(name, ".yaml")
				outPath := filepath.Join(outputDir, appName)
				exists, err := util.FileExists(outPath)
				if err != nil {
					return err
				}
				if !exists && strings.HasSuffix(appName, "-deployment") {
					// lets try strip "-deployment" from the file name
					appName = strings.TrimSuffix(appName, "-deployment")
					outPath = filepath.Join(outputDir, appName)
					exists, err = util.FileExists(outPath)
					if err != nil {
						return err
					}
				}
				if exists {
					chartDir := filepath.Join(outPath, "charts", appName)
					templatesDir := filepath.Join(chartDir, "templates")
					err = os.MkdirAll(templatesDir, DefaultWritePermissions)
					if err != nil {
						return err
					}

					valuesYaml := `replicaCount: 1`
					chartYaml := `apiVersion: v1
description: A Helm chart for Kubernetes
icon: https://raw.githubusercontent.com/jenkins-x/jenkins-x-platform/master/images/java.png
name: ` + appName + `
version: 0.0.1-SNAPSHOT
`
					helmIgnore := `# Patterns to ignore when building packages.
# This supports shell glob matching, relative path matching, and
# negation (prefixed with !). Only one pattern per line.
.DS_Store
# Common VCS dirs
.git/
.gitignore
.bzr/
.bzrignore
.hg/
.hgignore
.svn/
# Common backup files
*.swp
*.bak
*.tmp
*~
# Various IDEs
.project
.idea/
*.tmproj`

					err = generateFileIfMissing(filepath.Join(chartDir, "values.yaml"), valuesYaml)
					if err != nil {
						return err
					}
					err = generateFileIfMissing(filepath.Join(chartDir, "Chart.yaml"), chartYaml)
					if err != nil {
						return err
					}
					err = generateFileIfMissing(filepath.Join(chartDir, ".helmignore"), helmIgnore)
					if err != nil {
						return err
					}

					yaml, err := ioutil.ReadFile(path)
					if err != nil {
						return err
					}
					err = generateFileIfMissing(filepath.Join(templatesDir, "deployment.yaml"), string(yaml))
					if err != nil {
						return err
					}
				}

			}
		}
	}
	return nil
}

func (o *CommonOptions) createGitProviderForURL(gitKind string, gitUrl string) (gits.GitProvider, error) {
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return nil, err
	}
	return gits.CreateProviderForURL(o.Factory.IsInCluster(), authConfigSvc, gitKind, gitUrl, o.Git(), o.BatchMode, o.In, o.Out, o.Err)
}

func (o *CommonOptions) createGitProviderForURLWithoutKind(gitUrl string) (gits.GitProvider, *gits.GitRepositoryInfo, error) {
	gitInfo, err := gits.ParseGitURL(gitUrl)
	if err != nil {
		return nil, gitInfo, err
	}
	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return nil, gitInfo, err
	}
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return nil, gitInfo, err
	}
	gitProvider, err := gits.CreateProviderForURL(o.Factory.IsInCluster(), authConfigSvc, gitKind, gitInfo.HostURL(), o.Git(), o.BatchMode, o.In, o.Out, o.Err)
	return gitProvider, gitInfo, err
}

// generateFileIfMissing generates the given file from the source code if the file does not already exist
func generateFileIfMissing(path string, text string) error {
	exists, err := util.FileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return ioutil.WriteFile(path, []byte(text), DefaultWritePermissions)
	}
	return nil
}
