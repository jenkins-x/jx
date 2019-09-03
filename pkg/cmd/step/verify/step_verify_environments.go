package verify

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/jenkins-x/jx/pkg/cmd/opts/step"

	"github.com/jenkins-x/jx/pkg/boot"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// StepVerifyEnvironmentsOptions contains the command line flags
type StepVerifyEnvironmentsOptions struct {
	StepVerifyOptions
	Dir            string
	EnvDir         string
	LazyCreate     bool
	LazyCreateFlag string
}

// NewCmdStepVerifyEnvironments creates the `jx step verify pod` command
func NewCmdStepVerifyEnvironments(commonOpts *opts.CommonOptions) *cobra.Command {

	options := &StepVerifyEnvironmentsOptions{
		StepVerifyOptions: StepVerifyOptions{
			StepOptions: step.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "environments",
		Aliases: []string{"environment", "env"},
		Short:   "Verifies that the Environments have valid git repositories setup - lazily creating them if needed",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.LazyCreateFlag, "lazy-create", "", "", fmt.Sprintf("Specify true/false as to whether to lazily create missing resources. If not specified it is enabled if Terraform is not specified in the %s file", config.RequirementsConfigFileName))
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "the directory to look for the install requirements file, by default the current working directory")
	cmd.Flags().StringVarP(&options.EnvDir, "env-dir", "", "env", "the directory to look for the install requirements file relative to dir")
	return cmd
}

// Run implements this command
func (o *StepVerifyEnvironmentsOptions) Run() error {
	lazyCreate := true
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	requirements, _, err := config.LoadRequirementsConfig(filepath.Join(o.Dir, o.EnvDir))
	if err != nil {
		return err
	}
	info := util.ColorInfo

	envMap, names, err := kube.GetEnvironments(jxClient, ns)
	if err != nil {
		return errors.Wrapf(err, "failed to load Environments in namespace %s", ns)
	}
	for _, name := range names {
		env := envMap[name]
		gitURL := env.Spec.Source.URL
		if gitURL != "" && (env.Spec.Kind == v1.EnvironmentKindTypePermanent || (env.Spec.Kind == v1.EnvironmentKindTypeDevelopment && requirements.GitOps)) {
			log.Logger().Infof("validating git repository for %s at URL %s\n", info(name), info(gitURL))

			err = o.validateGitRepository(name, requirements, env, gitURL, lazyCreate)
			if err != nil {
				return err
			}
		}
	}

	log.Logger().Infof("the git repositories for the environments look good\n")
	fmt.Println()
	return nil
}

func (o *StepVerifyEnvironmentsOptions) prDevEnvironment(gitRepoName string, environmentsOrg string, privateRepo bool, user *auth.UserAuth, requirements *config.RequirementsConfig, server *auth.AuthServer, createPr bool) error {
	fromGitURL := os.Getenv("REPO_URL")
	gitRef := os.Getenv("BASE_CONFIG_REF")

	log.Logger().Debugf("Defined REPO_URL env variable value: %s", fromGitURL)
	log.Logger().Debugf("Defined BASE_CONFIG_REF env variable value: %s", gitRef)

	gitInfo, err := gits.ParseGitURL(fromGitURL)
	if err != nil {
		return errors.Wrapf(err, "parsing %s", fromGitURL)
	}

	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return errors.Wrapf(err, "getting server kind for %s", fromGitURL)
	}
	provider, err := gitInfo.CreateProviderForUser(server, user, gitKind, o.Git())
	if err != nil {
		return errors.Wrapf(err, "getting git provider for %s", fromGitURL)
	}
	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "resolving %s to absolute path", o.Dir)
	}

	if fromGitURL == config.DefaultBootRepository && gitRef == "master" {
		// If the GitURL is not overridden and the GitRef is set to it's default value then look up the version number
		resolver, err := o.CreateVersionResolver(requirements.VersionStream.URL, requirements.VersionStream.Ref)
		if err != nil {
			return errors.Wrapf(err, "failed to create version resolver")
		}
		gitRef, err = resolver.ResolveGitVersion(fromGitURL)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve version for https://github.com/jenkins-x/jenkins-x-boot-config.git")
		}
		if gitRef == "" {
			log.Logger().Infof("Attempting to resolve version for upstream boot config %s", util.ColorInfo(config.DefaultBootRepository))
			gitRef, err = resolver.ResolveGitVersion(config.DefaultBootRepository)
			if err != nil {
				return errors.Wrapf(err, "failed to resolve version for https://github.com/jenkins-x/jenkins-x-boot-config.git")
			}
		}
	}

	commitish, err := gits.FindTagForVersion(dir, gitRef, o.Git())
	if err != nil {
		log.Logger().Debugf(errors.Wrapf(err, "finding tag for %s", gitRef).Error())
		commitish = fmt.Sprintf("%s/%s", "origin", gitRef)
	}

	// Duplicate the repo
	duplicateInfo, err := gits.DuplicateGitRepoFromCommitish(environmentsOrg, gitRepoName, fromGitURL, commitish, "master", privateRepo, provider, o.Git())
	if err != nil {
		return errors.Wrapf(err, "duplicating %s to %s/%s", fromGitURL, environmentsOrg, gitRepoName)
	}

	_, baseRef, upstreamInfo, forkInfo, err := gits.ForkAndPullRepo(duplicateInfo.CloneURL, dir, "master", "master", provider, o.Git(), gitRepoName)
	if err != nil {
		return errors.Wrapf(err, "forking and pulling %s", duplicateInfo.CloneURL)
	}

	err = modifyPipelineGitEnvVars(dir)
	if err != nil {
		return errors.Wrap(err, "failed to modify dev environment config")
	}

	// Add a remote for the user that references the boot config that they originally used
	err = o.Git().SetRemoteURL(dir, "jenkins-x", fromGitURL)
	if err != nil {
		return errors.Wrapf(err, "Setting jenkins-x remote to boot config %s", fromGitURL)
	}

	if createPr {
		details := gits.PullRequestDetails{
			BranchName: fmt.Sprintf("update-boot-config"),
			Title:      "chore(config): update configuration",
			Message:    "chore(config): update configuration",
		}

		filter := gits.PullRequestFilter{
			Labels: []string{
				boot.PullRequestLabel,
			},
		}

		_, err = gits.PushRepoAndCreatePullRequest(dir, upstreamInfo, forkInfo, baseRef, &details, &filter, true, "chore(config): update configuration", true, false, o.Git(), provider, []string{boot.PullRequestLabel})
		if err != nil {
			return errors.Wrapf(err, "failed to create PR for base %s and head branch %s", baseRef, details.BranchName)
		}
	}
	return nil
}

func modifyPipelineGitEnvVars(dir string) error {
	parameterValues, err := helm.LoadParametersValuesFile(dir)
	if err != nil {
		return errors.Wrap(err, "failed to load parameters values file")
	}
	username := util.GetMapValueAsStringViaPath(parameterValues, "pipelineUser.username")
	email := util.GetMapValueAsStringViaPath(parameterValues, "pipelineUser.email")

	if username != "" && email != "" {
		fileName := filepath.Join(dir, config.ProjectConfigFileName)
		projectConf, err := config.LoadProjectConfigFile(fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to load project config file %s", fileName)
		}
		gitConfig := []corev1.EnvVar{
			{
				Name:  "GIT_AUTHOR_NAME",
				Value: username,
			},
			{
				Name:  "GIT_AUTHOR_EMAIL",
				Value: email,
			},
		}
		envVars := projectConf.PipelineConfig.Pipelines.Release.Pipeline.Environment
		envVars = append(envVars, gitConfig...)
		projectConf.PipelineConfig.Pipelines.Release.Pipeline.Environment = envVars

		err = projectConf.SaveConfig(fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to write to %s", fileName)
		}

		err = os.Setenv("GIT_AUTHOR_NAME", username)
		if err != nil {
			return errors.Wrap(err, "failed to set GIT_AUTHOR_NAME env variable")
		}
		err = os.Setenv("GIT_AUTHOR_EMAIL", email)
		if err != nil {
			return errors.Wrap(err, "failed to set GIT_AUTHOR_EMAIL env variable")
		}
	}
	return nil
}

func (o *StepVerifyEnvironmentsOptions) validateGitRepository(name string, requirements *config.RequirementsConfig, environment *v1.Environment, gitURL string, lazyCreate bool) error {
	message := fmt.Sprintf("for environment %s", environment.Name)
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse git URL %s and %s", gitURL, message)
	}
	authConfigSvc, err := o.CreatePipelineUserGitAuthConfigService()
	if err != nil {
		return err
	}
	return o.createEnvGitRepository(name, requirements, authConfigSvc, environment, gitURL, gitInfo)
}

func (o *StepVerifyEnvironmentsOptions) createEnvGitRepository(name string, requirements *config.RequirementsConfig, authConfigSvc auth.ConfigService, environment *v1.Environment, gitURL string, gitInfo *gits.GitRepository) error {
	log.Logger().Infof("creating environment %s git repository for URL: %s to namespace %s\n", util.ColorInfo(environment.Name), util.ColorInfo(gitURL), util.ColorInfo(environment.Spec.Namespace))

	envDir, err := ioutil.TempDir("", "jx-env-repo-")
	if err != nil {
		return err
	}

	// TODO
	gitKind := gits.KindGitHub
	publicRepo := requirements.Cluster.EnvironmentGitPublic
	prefix := ""

	gitServerURL := gitInfo.HostURL()
	server, userAuth := authConfigSvc.Config().GetPipelineAuth()
	helmValues, err := o.createEnvironmentHelmValues(requirements, environment)
	if err != nil {
		return err
	}
	batchMode := o.BatchMode
	forkGitURL := kube.DefaultEnvironmentGitRepoURL

	if server == nil {
		return fmt.Errorf("no auth server found for git server %s from gitURL %s", gitServerURL, gitURL)
	}
	if userAuth == nil {
		return fmt.Errorf("no pipeline user found for git server %s from gitURL %s", gitServerURL, gitURL)
	}
	if userAuth.IsInvalid() {
		return errors.Wrapf(err, "validating user '%s' of server '%s'", userAuth.Username, server.Name)
	}

	if name == kube.LabelValueDevEnvironment || environment.Spec.Kind == v1.EnvironmentKindTypeDevelopment {
		if requirements.GitOps {
			createPr := os.Getenv("JX_INTERPRET_PIPELINE") == "true"
			err := o.prDevEnvironment(gitInfo.Name, gitInfo.Organisation, !publicRepo, userAuth, requirements, server, createPr)
			if err != nil {
				return errors.Wrapf(err, "creating dev environment for %s", gitInfo.Name)
			}
		}
	} else {
		gitRepoOptions := &gits.GitRepositoryOptions{
			ServerURL:                gitServerURL,
			ServerKind:               gitKind,
			Username:                 userAuth.Username,
			ApiToken:                 userAuth.Password,
			Owner:                    gitInfo.Organisation,
			RepoName:                 gitInfo.Name,
			Public:                   publicRepo,
			IgnoreExistingRepository: true,
		}

		_, _, err = kube.DoCreateEnvironmentGitRepo(batchMode, authConfigSvc, environment, forkGitURL, envDir, gitRepoOptions, helmValues, prefix, o.Git(), o.ResolveChartMuseumURL, o.In, o.Out, o.Err)
		if err != nil {
			return errors.Wrapf(err, "failed to create git repository for gitURL %s", gitURL)
		}
	}
	return nil
}

func (o *StepVerifyEnvironmentsOptions) createEnvironmentHelmValues(requirements *config.RequirementsConfig, environment *v1.Environment) (config.HelmValuesConfig, error) {
	envCfg, err := requirements.Environment(environment.GetName())
	if err != nil || envCfg == nil {
		return config.HelmValuesConfig{}, errors.Wrapf(err,
			"looking the configuration of environment %q in the requirements configuration", environment.GetName())
	}
	domain := requirements.Ingress.Domain
	if envCfg.Ingress.Domain != "" {
		domain = envCfg.Ingress.Domain
	}
	useHTTP := "true"
	tlsAcme := "false"
	if envCfg.Ingress.TLS.Enabled {
		useHTTP = "false"
		tlsAcme = "true"
	}
	exposer := "Ingress"
	helmValues := config.HelmValuesConfig{
		ExposeController: &config.ExposeController{
			Config: config.ExposeControllerConfig{
				Domain:  domain,
				Exposer: exposer,
				HTTP:    useHTTP,
				TLSAcme: tlsAcme,
			},
		},
	}
	return helmValues, nil
}
