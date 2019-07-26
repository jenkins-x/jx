package verify

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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
			StepOptions: opts.StepOptions{
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

func (o *StepVerifyEnvironmentsOptions) prDevEnvironment(gitRepoName string, environmentsOrg string, server *auth.AuthServer, user *auth.UserAuth, requirements *config.RequirementsConfig) error {
	gitURL := os.Getenv("REPO_URL")
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return errors.Wrapf(err, "parsing %s", gitURL)
	}

	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return errors.Wrapf(err, "getting server kind for %s", gitURL)
	}
	provider, err := gitInfo.CreateProviderForUser(server, user, gitKind, o.Git())
	if err != nil {
		return errors.Wrapf(err, "getting git provider for %s", gitURL)
	}
	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "resolving %s to absolute path", o.Dir)
	}

	vs := requirements.VersionStream
	u := vs.URL
	ref := vs.Ref

	resolver, err := o.CreateVersionResolver(u, ref)
	if err != nil {
		return errors.Wrapf(err, "failed to create version resolver")
	}

	version, err := resolver.ResolveGitVersion("https://github.com/jenkins-x/jenkins-x-boot-config.git")
	if err != nil {
		return errors.Wrapf(err, "failed to resolve version for https://github.com/jenkins-x/jenkins-x-boot-config.git")
	}

	commitish, err := gits.FindTagForVersion(dir, version, o.Git())
	if err != nil {
		return errors.Wrapf(err, "finding tag for %s", version)
	}
	if commitish == "" {
		commitish = "master"
	}

	// Duplicate the repo
	duplicateInfo, err := gits.DuplicateGitRepoFromCommitsh(environmentsOrg, gitRepoName, gitURL, commitish, "master", o.Git(), provider)
	if err != nil {
		return errors.Wrapf(err, "duplicating %s to %s/%s", gitURL, environmentsOrg, gitRepoName)
	}

	_, baseRef, upstreamInfo, forkInfo, err := gits.ForkAndPullRepo(duplicateInfo.CloneURL, dir, "master", "master", provider, o.Git(), gitRepoName)
	if err != nil {
		return errors.Wrapf(err, "forking and pulling %s", duplicateInfo.CloneURL)
	}

	details := gits.PullRequestDetails{
		BranchName: fmt.Sprintf("update-boot-config"),
		Title:      "chore(config): update configuration",
		Message:    "chore(config): update configuration",
	}

	_, err = gits.PushRepoAndCreatePullRequest(dir, upstreamInfo, forkInfo, baseRef, &details, nil, true, "chore(config): update configuration", true, false, false, o.Git(), provider)
	if err != nil {
		return errors.Wrapf(err, "failed to create PR for base %s and head branch %s", baseRef, details.BranchName)
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
	privateRepo := false
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
		err := o.prDevEnvironment(gitInfo.Name, gitInfo.Organisation, server, userAuth, requirements)
		if err != nil {
			return errors.Wrapf(err, "creating dev environment for %s", gitInfo.Name)
		}
	} else {
		gitRepoOptions := &gits.GitRepositoryOptions{
			ServerURL:                gitServerURL,
			ServerKind:               gitKind,
			Username:                 userAuth.Username,
			ApiToken:                 userAuth.Password,
			Owner:                    gitInfo.Organisation,
			RepoName:                 gitInfo.Name,
			Private:                  privateRepo,
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
	// lets default the ingress requirements
	domain := requirements.Ingress.Domain
	useHTTP := "true"
	tlsAcme := ""
	if requirements.Ingress.TLS.Enabled {
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
