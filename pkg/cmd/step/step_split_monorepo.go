package step

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/opts/step"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

const (
	optionOrganisation   = "organisation"
	defaultKubernetesDir = "kubernetes"
)

var (
	stepSplitMonorepoLong = templates.LongDesc(`
		Mirrors the code from a monorepo into separate microservice style Git repositories so its easier to do finer grained releases.

		If you have lots of apps in folders in a monorepo then this command can run on that repo to mirror changes into a number of microservice based repositories which can each then get auto-imported into Jenkins X

`)

	stepSplitMonorepoExample = templates.Examples(`
		# Split the current folder up into separate Git repositories 
		jx step split -o mygithuborg
			`)
)

// StepSplitMonorepoOptions contains the command line flags
type StepSplitMonorepoOptions struct {
	step.StepOptions
	gits.GitRepositoryOptions

	Glob          string
	Organisation  string
	Dir           string
	OutputDir     string
	KubernetesDir string
	NoGit         bool
}

// NewCmdStepSplitMonorepo Creates a new Command object
func NewCmdStepSplitMonorepo(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepSplitMonorepoOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "split",
		Short:   "Mirrors the code from a monorepo into separate microservice style Git repositories so its easier to do finer grained releases",
		Long:    stepSplitMonorepoLong,
		Example: stepSplitMonorepoExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Glob, "glob", "g", "*", "The glob pattern to find folders to mirror to separate repositories")
	cmd.Flags().StringVarP(&options.Organisation, optionOrganisation, "o", "", "The GitHub organisation to split the repositories into")
	cmd.Flags().StringVarP(&options.Dir, "source-dir", "s", "", "The source directory to look inside for the folders to move into separate Git repositories")
	cmd.Flags().StringVarP(&options.OutputDir, opts.OptionOutputDir, "d", "generated", "The output directory where new projects are created")
	cmd.Flags().StringVarP(&options.KubernetesDir, "kubernetes-folder", "", defaultKubernetesDir, "The folder containing all the Kubernetes YAML for each app")
	cmd.Flags().BoolVarP(&options.NoGit, "no-git", "", false, "If enabled then don't create and push to new repositories at remote git provider")

	opts.AddGitRepoOptionsArguments(cmd, &options.GitRepositoryOptions)

	return cmd
}

// Run implements this command
func (options *StepSplitMonorepoOptions) Run() error {
	err := options.DefaultsFromTeamSettings()
	if err != nil {
		return err
	}
	if options.Organisation == "" {
		return util.MissingOption(optionOrganisation)
	}
	if options.OutputDir == "" {
		return util.MissingOption(opts.OptionOutputDir)
	}
	dir := options.Dir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	glob := options.Glob

	fullGlob := filepath.Join(dir, glob)
	log.Logger().Debugf("Searching in monorepo at: %s", fullGlob)
	matches, err := filepath.Glob(fullGlob)
	if err != nil {
		return err
	}
	kubeDir := options.KubernetesDir
	if kubeDir == "" {
		kubeDir = defaultKubernetesDir
	}
	gitDir, err := options.Git().RevParse(dir, "--show-toplevel")
	if err != nil {
		return err
	}
	currentBranch, err := options.Git().Branch(gitDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(options.OutputDir, util.DefaultWritePermissions)
	if err != nil {
		return err
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
				log.Logger().Debugf("Found match: %s", path)
				outPath := filepath.Join(options.OutputDir, name)
				err = options.Git().LocalClone(gitDir, options.OutputDir, name, currentBranch)
				if err != nil {
					return err
				}
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				pathInRepo := strings.TrimPrefix(strings.TrimPrefix(absPath, gitDir), string(filepath.Separator))
				err = options.Git().FilterBranch(outPath, pathInRepo, currentBranch) // Use current branch
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
					rootGitIgnore := filepath.Join(gitDir, ".gitignore")
					exists, err = util.FileExists(rootGitIgnore)
					if err != nil {
						return err
					}
					if exists {
						err = util.CopyFile(rootGitIgnore, localGitIgnore)
						if err != nil {
							return err
						}
						options.Git().Add(outPath, ".gitignore")
					}
				}

				if !options.NoGit {
					authConfigSvc, err := options.GitLocalAuthConfigService()
					if err != nil {
						return err
					}
					options.GitRepositoryOptions.Owner = options.Organisation
					options.GitRepositoryOptions.RepoName = name
					details, err := gits.PickNewOrExistingGitRepository(options.BatchMode, authConfigSvc,
						"", &options.GitRepositoryOptions, nil, nil, options.Git(), false, options.GetIOFileHandles())
					if err != nil {
						return err
					}
					repo, err := details.CreateRepository()
					if err != nil {
						return err
					}

					log.Logger().Infof("Created Git repository to %s\n", util.ColorInfo(repo.HTMLURL))
					gitURL, err := options.Git().CreateAuthenticatedURL(repo.CloneURL, details.User)

					err = options.Git().Init(outPath)
					if err != nil {
						return err
					}
					err = options.Git().AddRemote(outPath, "origin", gitURL)
					if err != nil {
						return err
					}

					message := "generated by: jx step split monorepo"
					err = options.Git().CommitIfChanges(outPath, message)
					if err != nil {
						return err
					}
					err = options.Git().PushMaster(outPath)
					if err != nil {
						return err
					}
					log.Logger().Infof("Pushed Git repository to %s\n", util.ColorInfo(repo.HTMLURL))
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
				outPath := filepath.Join(options.OutputDir, appName)
				exists, err := util.FileExists(outPath)
				if err != nil {
					return err
				}
				if !exists && strings.HasSuffix(appName, "-deployment") {
					// lets try strip "-deployment" from the file name
					appName = strings.TrimSuffix(appName, "-deployment")
					outPath = filepath.Join(options.OutputDir, appName)
					exists, err = util.FileExists(outPath)
					if err != nil {
						return err
					}
				}
				if exists {
					chartDir := filepath.Join(outPath, "charts", appName)
					templatesDir := filepath.Join(chartDir, "templates")
					err = os.MkdirAll(templatesDir, util.DefaultWritePermissions)
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

// generateFileIfMissing generates the given file from the source code if the file does not already exist
func generateFileIfMissing(path string, text string) error {
	exists, err := util.FileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return ioutil.WriteFile(path, []byte(text), util.DefaultWritePermissions)
	}
	return nil
}

// DefaultsFromTeamSettings fills in options for team settings
func (options *StepSplitMonorepoOptions) DefaultsFromTeamSettings() error {
	settings, err := options.TeamSettings()
	if err != nil {
		return err
	}
	if options.Organisation == "" {
		options.Organisation = settings.Organisation
	}
	if options.GitRepositoryOptions.ServerURL == "" {
		options.GitRepositoryOptions.ServerURL = settings.GitServer
	}
	options.GitRepositoryOptions.Public = settings.GitPublic || options.GitRepositoryOptions.Public
	return nil
}
