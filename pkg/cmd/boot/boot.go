package boot

import (
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	Dir       string
	GitURL    string
	GitRef    string
	StartStep string

	// The bootstrap URL for the version stream. Once we have a jx-requirements.yaml files, we read that
	VersionStreamURL string
	// The bootstrap ref for the version stream. Once we have a jx-requirements.yaml, we read that
	VersionStreamRef string
}

var (
	bootLong = templates.LongDesc(`
		Boots up Jenkins X in a Kubernetes cluster using GitOps and a Jenkins X Pipeline

		For more documentation see: [https://jenkins-x.io/getting-started/boot/](https://jenkins-x.io/getting-started/boot/)

`)

	bootExample = templates.Examples(`
		# create a kubernetes cluster via Terraform or via jx
		jx create cluster gke --skip-installation

		# now lets boot up Jenkins X installing/upgrading whatever is needed
		jx boot 

		# if we have already booted and just want to apply some environment changes without 
        # re-applying ingress and so forth we can start at the environment step:
		jx boot --start-step install-env
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
	cmd.Flags().StringVarP(&options.GitURL, "git-url", "u", "", "override the Git clone URL for the JX Boot source to start from, ignoring the versions stream. Normally specified with git-ref as well")
	cmd.Flags().StringVarP(&options.GitRef, "git-ref", "", "", "override the Git ref for the JX Boot source to start from, ignoring the versions stream. Normally specified with git-url as well")
	cmd.Flags().StringVarP(&options.VersionStreamURL, "versions-repo", "", config.DefaultVersionsURL, "the bootstrap URL for the versions repo. Once the boot config is cloned, the repo will be then read from the jx-requirements.yaml")
	cmd.Flags().StringVarP(&options.VersionStreamRef, "versions-ref", "", config.DefaultVersionsRef, "the bootstrap ref for the versions repo. Once the boot config is cloned, the repo will be then read from the jx-requirements.yaml")
	cmd.Flags().StringVarP(&options.StartStep, "start-step", "s", "", "the step in the pipeline to start from")
	return cmd
}

// Run runs this command
func (o *BootOptions) Run() error {
	info := util.ColorInfo

	err := o.verifyClusterConnection()
	if err != nil {
		return err
	}

	_, pipelineFile, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		return err
	}
	exists, err := util.FileExists(pipelineFile)
	if err != nil {
		return err
	}

	gitURL, gitRef, err := gits.GetGitInfoFromDirectory(o.Dir, o.Git())
	if err != nil {
		log.Logger().Warn(fmt.Sprintf("there was a problem obtaining the boot config repository git configuration, falling back to defaults"))
		gitURL = config.DefaultBootRepository
		gitRef = config.DefaultVersionsRef
	}

	if o.GitURL != "" {
		log.Logger().Infof("GitURL provided, overriding the current value: %s", util.ColorInfo(gitURL))
		gitURL = o.GitURL
	}

	if o.GitRef != "" {
		log.Logger().Infof("GitRef provided, overriding the current value: %s", util.ColorInfo(gitRef))
		gitRef = o.GitRef
	}

	if config.LoadActiveInstallProfile() == config.CloudBeesProfile && o.GitURL == "" {
		gitURL = config.DefaultCloudBeesBootRepository
	}
	if config.LoadActiveInstallProfile() == config.CloudBeesProfile && o.VersionStreamURL == config.DefaultVersionsURL {
		o.VersionStreamURL = config.DefaultCloudBeesVersionsURL

	}
	if config.LoadActiveInstallProfile() == config.CloudBeesProfile && o.VersionStreamRef == config.DefaultVersionsRef {
		o.VersionStreamRef = config.DefaultCloudBeesVersionsRef

	}
	if gitURL == "" {
		return util.MissingOption("git-url")
	}

	requirements, requirementsFile, _ := config.LoadRequirementsConfig(o.Dir)
	// lets report errors parsing this file after the check we are outside of a git clone
	o.defaultVersionStream(requirements)

	if !exists {
		log.Logger().Infof("No Jenkins X pipeline file %s found. You are not running this command from inside a Jenkins X Boot git clone", info(pipelineFile))

		gitInfo, err := gits.ParseGitURL(gitURL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse git URL %s", gitURL)
		}

		repo := gitInfo.Name
		cloneDir := filepath.Join(o.Dir, repo)

		if o.GitURL == "" && o.GitRef == "" {
			// If the GitURL is not overridden and the GitRef is set to it's default value then look up the version number
			resolver, err := o.CreateVersionResolver(requirements.VersionStream.URL, requirements.VersionStream.Ref)
			if err != nil {
				return errors.Wrapf(err, "failed to create version resolver")
			}
			gitRef, err = resolver.ResolveGitVersion(gitURL)
			if err != nil {
				return errors.Wrapf(err, fmt.Sprintf("failed to resolve version for %s", gitURL))
			}
			if gitRef == "" {
				log.Logger().Infof("Attempting to resolve version for upstream boot config %s", util.ColorInfo(config.DefaultBootRepository))
				gitRef, err = resolver.ResolveGitVersion(config.DefaultBootRepository)
				if err != nil {
					return errors.Wrapf(err, fmt.Sprintf("failed to resolve version for %s", config.DefaultBootRepository))
				}
			}

		}

		if !o.BatchMode {
			log.Logger().Infof("To continue we will clone %s @ %s to %s", info(gitURL), info(gitRef), info(cloneDir))

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

		log.Logger().Infof("Cloning %s @ %s to %s\n", info(gitURL), info(gitRef), info(cloneDir))

		err = os.MkdirAll(cloneDir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create directory: %s", cloneDir)
		}

		err = o.Git().Clone(gitURL, cloneDir)
		if err != nil {
			return errors.Wrapf(err, "failed to clone git URL %s to directory: %s", gitURL, cloneDir)
		}
		commitish, err := gits.FindTagForVersion(cloneDir, gitRef, o.Git())
		if err != nil {
			log.Logger().Debugf(errors.Wrapf(err, "finding tag for %s", o.GitRef).Error())
			commitish = fmt.Sprintf("%s/%s", "origin", gitRef)
		}
		err = o.Git().Reset(cloneDir, commitish, true)
		if err != nil {
			return errors.Wrapf(err, "setting HEAD to %s", commitish)
		}
		o.Dir, err = filepath.Abs(cloneDir)
		if err != nil {
			return err
		}

		_, pipelineFile, err = config.LoadProjectConfig(o.Dir)
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

	requirements, requirementsFile, err = config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return errors.Wrap(err, "failed to load jx-requirements.yml file")
	}
	o.defaultVersionStream(requirements)

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
	so.StartStep = o.StartStep
	so.CommonOptions.SetDevNamespace(requirements.Cluster.Namespace)
	so.AdditionalEnvVars = map[string]string{
		"JX_NO_TILLER":     "true",
		"REPO_URL":         gitURL,
		"BASE_CONFIG_REF":  gitRef,
		"DEPLOY_NAMESPACE": requirements.Cluster.Namespace,
	}

	so.VersionResolver, err = o.CreateVersionResolver(requirements.VersionStream.URL, requirements.VersionStream.Ref)
	if err != nil {
		return errors.Wrapf(err, "there was a problem creating a version resolver from versions stream repository %s and ref %s", requirements.VersionStream.URL, requirements.VersionStream.Ref)
	}

	if o.BatchMode {
		so.AdditionalEnvVars["JX_BATCH_MODE"] = "true"
	}
	err = so.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to interpret pipeline file %s", pipelineFile)
	}

	// lets switch kubernetes context to it so the user can use `jx` commands immediately
	no := &namespace.NamespaceOptions{}
	no.CommonOptions = o.CommonOptions
	no.Args = []string{requirements.Cluster.Namespace}
	log.Logger().Infof("switching to the namespace %s so that you can use %s commands on the installation", info(requirements.Cluster.Namespace), info("jx"))
	return no.Run()
	return nil
}

func (o *BootOptions) defaultVersionStream(requirements *config.RequirementsConfig) {
	if requirements.VersionStream.URL == "" && requirements.VersionStream.Ref == "" {
		requirements.VersionStream.URL = o.VersionStreamURL
		requirements.VersionStream.Ref = o.VersionStreamRef
	}
	// If we still don't have a complete version stream ref then we better set to a default
	if requirements.VersionStream.URL == "" || requirements.VersionStream.Ref == "" {
		log.Logger().Warnf("Incomplete version stream reference %s @ %s", requirements.VersionStream.URL, requirements.VersionStream.Ref)
		if config.LoadActiveInstallProfile() == config.CloudBeesProfile {
			o.VersionStreamRef = config.DefaultCloudBeesVersionsRef
			o.VersionStreamURL = config.DefaultVersionsURL
		} else {
			o.VersionStreamRef = config.DefaultVersionsRef
			o.VersionStreamURL = config.DefaultVersionsURL
		}
		log.Logger().Infof("Setting version stream reference to default %s @ %s", requirements.VersionStream.URL, requirements.VersionStream.Ref)
	}
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
	if requirements.Cluster.Namespace == "" {
		return config.MissingRequirement("namespace", requirementsFile)
	}
	return nil
}

func (o *BootOptions) verifyClusterConnection() error {
	client, err := o.KubeClient()
	if err == nil {
		_, err = client.CoreV1().Namespaces().List(metav1.ListOptions{})
	}

	if err != nil {
		return fmt.Errorf("You are not currently connected to a cluster, please connect to the cluster that you intend to %s\n"+
			"Alternatively create a new cluster using %s", util.ColorInfo("jx boot"), util.ColorInfo("jx create cluster"))
	}
	return nil
}
