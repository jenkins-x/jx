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

// BootOptions options for the command
type BootOptions struct {
	*opts.CommonOptions

	Dir    string
	GitURL string

	// The bootstrap URL for the version stream. Once we have a jx-requirements.yaml files, we read that
	VersionStreamURL string
	// The bootstrap ref for the version stream. Once we have a jx-requirements.yaml, we read that
	VersionStreamRef string
}

var (
	bootLong = templates.LongDesc(`
		Boots up Jenkins X in a Kubernetes cluster using GitOps and a Jenkins X Pipeline

`)

	bootExample = templates.Examples(`
		# create a kubernetes cluster via Terraform or via jx
		jx create cluster gke --skip-installation

		# lets get the GitOps repository source code
		git clone https://github.com/jenkins-x/jenkins-x-boot-config.git my-jx-config
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
	cmd.Flags().StringVarP(&options.GitURL, "git-url", "u", config.DefaultBootRepository, "the Git clone URL for the JX Boot source to start from")
	cmd.Flags().StringVarP(&options.VersionStreamURL, "versions-repo", "", config.DefaultVersionsURL, "the bootstrap URL for the versions repo. Once the boot config is cloned, the repo will be then read from the jx-requirements.yaml")
	cmd.Flags().StringVarP(&options.VersionStreamRef, "versions-ref", "", config.DefaultVersionsRef, "the bootstrap ref for the versions repo. Once the boot config is cloned, the repo will be then read from the jx-requirements.yaml")
	return cmd
}

// Run runs this command
func (o *BootOptions) Run() error {
	info := util.ColorInfo

	err := o.verifyClusterConnection()
	if err != nil {
		return err
	}

	projectConfig, pipelineFile, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		return err
	}
	exists, err := util.FileExists(pipelineFile)
	if err != nil {
		return err
	}

	if config.LoadActiveInstallProfile() == config.CloudBeesProfile && o.GitURL == config.DefaultBootRepository {
		o.GitURL = config.DefaultCloudBeesBootRepository

	}
	if config.LoadActiveInstallProfile() == config.CloudBeesProfile && o.VersionStreamURL == config.DefaultVersionsURL {
		o.VersionStreamURL = config.DefaultCloudBeesVersionsURL

	}
	if config.LoadActiveInstallProfile() == config.CloudBeesProfile && o.VersionStreamRef == config.DefaultVersionsRef {
		o.VersionStreamRef = config.DefaultCloudBeesVersionsRef

	}
	if o.GitURL == "" {
		return util.MissingOption("git-url")
	}

	if !exists {
		log.Logger().Infof("No Jenkins X pipeline file %s found. You are not running this command from inside a Jenkins X Boot git clone", info(pipelineFile))

		gitInfo, err := gits.ParseGitURL(o.GitURL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse git URL %s", o.GitURL)
		}

		repo := gitInfo.Name
		cloneDir := filepath.Join(o.Dir, repo)

		resolver, err := o.CreateVersionResolver(o.VersionStreamURL, o.VersionStreamRef)
		if err != nil {
			return errors.Wrapf(err, "failed to create version resolver")
		}

		version, err := resolver.ResolveGitVersion("https://github.com/jenkins-x/jenkins-x-boot-config.git")
		if err != nil {
			return errors.Wrapf(err, "failed to resolve version for https://github.com/jenkins-x/jenkins-x-boot-config.git")
		}

		if !o.BatchMode {
			log.Logger().Infof("To continue we will clone %s @ %s to %s", info(o.GitURL), info(version), info(cloneDir))

			help := "A git clone of a Jenkins X Boot source repository is required for 'jx boot'"
			message := "Do you want to clone the Jenkins X Boot Git repository?"
			if !util.Confirm(message, true, help, o.In, o.Out, o.Err) {
				return fmt.Errorf("Please run this command again inside a git clone from a Jenkins X Boot repository")
			}
		}

		exists, err = util.FileExists(cloneDir)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Cannot clone git repository to %s as the dir already exists. Maybe try 'cd %s' and re-run the 'jx boot' command?", repo, repo)
		}

		log.Logger().Infof("Cloning %s with version %s to %s\n", info(o.GitURL), info(version), info(cloneDir))

		err = os.MkdirAll(cloneDir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create directory: %s", cloneDir)
		}

		err = o.Git().Clone(o.GitURL, cloneDir)
		if err != nil {
			return errors.Wrapf(err, "failed to clone git URL %s to directory: %s", o.GitURL, cloneDir)
		}
		commitish, err := gits.FindTagForVersion(cloneDir, version, o.Git())
		if err != nil {
			return errors.Wrapf(err, "finding tag for %s", version)
		}
		if commitish == "" {
			commitish = "origin/master"
		}
		err = o.Git().ResetHard(cloneDir, commitish)
		if err != nil {
			return errors.Wrapf(err, "setting HEAD to %s", commitish)
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
			return fmt.Errorf("The cloned repository %s does not include a Jenkins X Pipeline file at %s", o.GitURL, pipelineFile)
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

	log.Logger().Infof("booting up Jenkins X")

	// now lets really boot
	_, so := create.NewCmdStepCreateTaskAndOption(o.CommonOptions)
	so.CloneDir = o.Dir
	so.CloneDir = o.Dir
	so.InterpretMode = true
	so.NoReleasePrepare = true
	so.AdditionalEnvVars = map[string]string{
		"JX_NO_TILLER": "true",
		"REPO_URL":     o.GitURL,
	}

	so.VersionResolver, err = o.CreateVersionResolver(requirements.VersionStream.URL, requirements.VersionStream.Ref)
	if err != nil {
		return errors.Wrapf(err, "there was a problem creating a version resolver from versions stream repository %s and ref %s", requirements.VersionStream.URL, requirements.VersionStream.Ref)
	}

	if o.BatchMode {
		so.AdditionalEnvVars["JX_BATCH_MODE"] = "true"
	}
	ns := FindBootNamespace(projectConfig, requirements)
	if ns != "" {
		so.CommonOptions.SetDevNamespace(ns)
	}
	err = so.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to interpret pipeline file %s", pipelineFile)
	}

	// if we can find the deploy namespace lets switch kubernetes context to it so the user can use `jx` commands immediately
	if ns != "" {
		no := &namespace.NamespaceOptions{}
		no.CommonOptions = o.CommonOptions
		no.Args = []string{ns}
		log.Logger().Infof("switching to the namespace %s so that you can use %s commands on the installation", info(ns), info("jx"))
		return no.Run()
	}
	return nil
}

func (o *BootOptions) verifyRequirements(requirements *config.RequirementsConfig, requirementsFile string) error {
	provider := requirements.Cluster.Provider
	if provider == "" {
		return config.MissingRequirement("provider", requirementsFile)
	}
	if provider == "" {
		if requirements.Cluster.ProjectID == "" {
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

func (o *BootOptions) verifyClusterConnection() error {
	_, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("You are not currently connected to a cluster, please connect to the cluster that you intend to %s\n"+
			"Alternatively create a new cluster using %s", util.ColorInfo("jx boot"), util.ColorInfo("jx create cluster"))
	}
	return nil
}
