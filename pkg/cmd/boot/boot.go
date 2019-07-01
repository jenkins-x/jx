package boot

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/namespace"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
)

const (
	defaultBootRepository = "https://github.com/jstrachan/environment-simple-tekton.git"
)

// BootOptions options for the command
type BootOptions struct {
	*opts.CommonOptions

	Dir    string
	GitURL string
}

var (
	bootLong = templates.LongDesc(`
		Boots up Jenkins X in a Kubernetes cluster using GitOps and a Jenkins X Pipeline

`)

	bootExample = templates.Examples(`
		# create a kubernetes cluster via Terraform or via jx
		jx create cluster gke --skip-installation

		# lets get the GitOps repository source code
		git clone https://github.com/jstrachan/environment-simple-tekton.git my-jx-config
		cd my-jx-config

		# now lets boot up Jenkins X installing/upgrading whatever is needed
		jx boot 
`)
)

// NewCmdBoot creates the command
func NewCmdBoot(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &BootOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "boot",
		Aliases: []string{"bootstrap"},
		Short:   "Boots up Jenkins X in a Kubernetes cluster using GitOps and a Jenkins X Pipeline",
		Long:    bootLong,
		Example: bootExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the Jenkins X Pipeline, requirements and charts")
	cmd.Flags().StringVarP(&options.GitURL, "git-url", "u", defaultBootRepository, "the Git clone URL for the JX Boot source to boot up")
	return cmd
}

// Run runs this command
func (o *BootOptions) Run() error {
	info := util.ColorInfo

	projectConfig, pipelineFile, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		return err
	}
	exists, err := util.FileExists(pipelineFile)
	if err != nil {
		return err
	}
	if !exists {
		log.Logger().Infof("No Jenkins X pipeline file %s found. You are not running this command from inside a Jenkins X Boot git clone\n\n", info(pipelineFile))

		gitURL := o.GitURL
		if gitURL == "" {
			return util.MissingOption("git-url")
		}
		gitInfo, err := gits.ParseGitURL(gitURL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse git URL %s", gitURL)
		}

		repo := gitInfo.Name
		cloneDir := filepath.Join(o.Dir, repo)

		if !o.BatchMode {
			log.Logger().Infof("To continue we will clone: %s\n", info(gitURL))
			log.Logger().Infof("To the directory: %s\n\n", info(cloneDir))

			help := "A git clone of a Jenkins X Boot source repository is required for 'jx boot'"
			message := "Do you want to clone the Jenkins X Boot Git repository?"
			if !util.Confirm(message, true, help, o.In, o.Out, o.Err) {
				return fmt.Errorf("Please run this command again inside a git clone from a Jenkins X Boot repository")
			}
		}

		exists, err = util.FileExists(pipelineFile)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Cannot clone git repository to %s as the dir already exists. Maybe try 'cd %s' and re-run the 'jx boot' command?", repo, repo)
		}

		log.Logger().Infof("\ncloning: %s to directory: %s\n", info(gitURL), info(cloneDir))

		err = os.MkdirAll(cloneDir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create directory: %s", cloneDir)
		}

		err = o.Git().Clone(gitURL, cloneDir)
		if err != nil {
			return errors.Wrapf(err, "failed to clone git URL %s to directory: %s", gitURL, cloneDir)
		}

		o.Dir, err = filepath.Abs(cloneDir)
		if err != nil {
			return err
		}

		projectConfig, pipelineFile, err = config.LoadProjectConfig(o.Dir)
		if err != nil {
			return err
		}
		exists, err = util.FileExists(pipelineFile)
		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf("The cloned repository %s does not include a Jenkins X Pipeline file at %s", gitURL, pipelineFile)
		}
	}

	requirements, requirementsFile, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return err
	}
	exists, err = util.FileExists(requirementsFile)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("No requirements file %s are you sure you are running this command inside a GitOps clone?", requirementsFile)
	}

	err = o.verifyRequirements(requirements, requirementsFile)
	if err != nil {
		return err
	}

	log.Logger().Infof("booting up Jenkins X\n")

	// now lets really boot
	_, so := create.NewCmdStepCreateTaskAndOption(o.CommonOptions)
	so.CloneDir = o.Dir
	so.CloneDir = o.Dir
	so.InterpretMode = true
	so.NoReleasePrepare = true
	so.AdditionalEnvVars = map[string]string{
		"JX_NO_TILLER": "true",
	}
	if o.BatchMode {
		so.AdditionalEnvVars["JX_BATCH_MODE"] = "true"
	}
	err = so.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to interpret pipeline file %s", pipelineFile)
	}

	// if we can find the deploy namespace lets switch kubernetes context to it so the user can use `jx` commands immediately
	ns := FindBootNamespace(projectConfig, requirements)
	if ns != "" {
		no := &namespace.NamespaceOptions{}
		no.CommonOptions = o.CommonOptions
		no.Args = []string{ns}
		log.Logger().Infof("switching to the namespace %s so that you can use %s commands on the installation\n", info(ns), info("jx"))
		return no.Run()
	}
	return nil
}

func (o *BootOptions) verifyRequirements(requirements *config.RequirementsConfig, requirementsFile string) error {
	provider := requirements.Provider
	if provider == "" {
		return config.MissingRequirement("provider", requirementsFile)
	}
	if provider == "" {
		if requirements.ProjectID == "" {
			return config.MissingRequirement("project", requirementsFile)
		}
	}
	return nil
}

// FindBootNamespace finds the namespace to boot Jenkins X into based on the pipeline and requirements
func FindBootNamespace(projectConfig *config.ProjectConfig, requirementsConfig *config.RequirementsConfig) string {
	// TODO should we add the deploy namepace to jx-requirements.yml?
	if projectConfig != nil {
		pipelineConfig := projectConfig.PipelineConfig
		if pipelineConfig != nil {
			release := pipelineConfig.Pipelines.Release
			if release != nil {
				pipeline := release.Pipeline
				if pipeline != nil {
					for _, env := range pipeline.Environment {
						if env.Name == "DEPLOY_NAMESPACE" && env.Value != "" {
							return env.Value
						}
					}
				}
			}
		}
	}
	return ""
}
