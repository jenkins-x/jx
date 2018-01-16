package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"path/filepath"
	"os"
	"github.com/jenkins-x/jx/pkg/helm"
)

const (
	optionEnvironment = "environment"
)

// PromoteOptions containers the CLI options
type PromoteOptions struct {
	CommonOptions

	Namespace         string
	Environment       string
	Application       string
	Version           string
	LocalHelmRepoName string
	Preview           bool
	NoHelmUpdate      bool
}

var (
	promote_long = templates.LongDesc(`
		Promotes a version of an application to an environment.
`)

	promote_example = templates.Examples(`
		# Promote a version of an application to staging
		jx promote myapp --version 1.2.3 --env staging
	`)
)

// NewCmdPromote creates the new command for: jx get prompt
func NewCmdPromote(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &PromoteOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "promote [application]",
		Short:   "Promotes a version of an application to an environment",
		Long:    promote_long,
		Example: promote_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The Namespace to promote to")
	cmd.Flags().StringVarP(&options.Environment, optionEnvironment, "e", "", "The Environment to promote to")
	cmd.Flags().StringVarP(&options.Application, "app", "a", "", "The Application to promote")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "The Version to promote")
	cmd.Flags().StringVarP(&options.LocalHelmRepoName, "helm-repo-name", "r", kube.LocalHelmRepoName, "The name of the helm repository that contains the app")
	cmd.Flags().BoolVarP(&options.Preview, "preview", "p", false, "Whether to create a new Preview environment for the app")
	cmd.Flags().BoolVarP(&options.NoHelmUpdate, "no-helm-update", "", false, "Allows the 'helm repo update' command if you are sure your local helm cache is up to date with the version you wish to promote")

	return cmd
}

// Run implements this command
func (o *PromoteOptions) Run() error {
	targetNS, env, err := o.GetTargetNamespace(o.Namespace, o.Environment)
	if err != nil {
		return err
	}
	app := o.Application
	if app == "" {
		args := o.Args
		if len(args) == 0 {
			return fmt.Errorf("Missing application argument")
		}
		app = args[0]
	}
	o.Application = app
	version := o.Version
	info := util.ColorInfo
	if version == "" {
		o.Printf("Promoting latest version of app %s to namespace %s\n", info(app), info(targetNS))
	} else {
		o.Printf("Promoting app %s version %s to namespace %s\n", info(app), info(version), info(targetNS))
	}

	if env != nil && env.Spec.PromotionStrategy == v1.PromotionStrategyTypeAutomatic {
		o.Printf("%s", util.ColorWarning("WARNING: The Environment %s is setup to promote automatically as part of the CI / CD Pipelines.\n\n", env.Name))

		confirm := &survey.Confirm{
			Message: "Do you wish to promote anyway? :",
			Default: false,
		}
		flag := false
		err := survey.AskOne(confirm, &flag, nil)
		if err != nil {
			return err
		}
		if !flag {
			return nil
		}
	}
	if env != nil {
		source := &env.Spec.Source
		if source.URL != "" {
			return o.PromoteViaPullRequest(env)
		}
	}
	fullAppName := app
	if o.LocalHelmRepoName != "" {
		fullAppName = o.LocalHelmRepoName + "/" + app
	}

	// lets do a helm update to ensure we can find the latest version
	if !o.NoHelmUpdate {
		o.Printf("Updading the helm repositories to ensure we can find the latest versions...")
		err = o.runCommand("helm", "repo", "update")
		if err != nil {
			return err
		}
	}
	if version != "" {
		return o.runCommand("helm", "install", "--namespace", targetNS, "--version", version, fullAppName)
	}
	return o.runCommand("helm", "install", "--namespace", targetNS, fullAppName)
}

func (o *PromoteOptions) PromoteViaPullRequest(env *v1.Environment) error {
	source := &env.Spec.Source
	gitURL := source.URL
	if gitURL == "" {
		return fmt.Errorf("No source git URL")
	}
	team, _, err := o.TeamAndEnvironmentNames()
	if err != nil {
		return err
	}

	environmentsDir, err := cmdutil.TeamEnvironmentsDir(team)
	if err != nil {
		return err
	}
	dir := filepath.Join(environmentsDir, env.Name)

	// now lets clone the fork and push it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return err
	}

	app := o.Application
	version := o.Version
	versionName := version
	if versionName == "" {
		versionName = "latest"
	}

	branchName := gits.ConvertToValidBranchName("promote-" + app + "-" + versionName)
	base := source.Ref
	if base == ""  {
		base = "master"
	}
	if exists {
		// lets make sure that the origin is correct...
		err = gits.GitCmd(dir, "remote", "add", "origin", gitURL)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "stash")
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "checkout", base)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "pull")
		if err != nil {
			return err
		}
	} else {
		err := os.MkdirAll(dir, DefaultWritePermissions)
		if err != nil {
			return fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		err = gits.GitClone(gitURL, dir)
		if err != nil {
			return err
		}
		if base != "master" {
			err = gits.GitCmd(dir, "checkout", base)
			if err != nil {
				return err
			}
		}

		// TODO lets fork if required???
		/*
		pushGitURL, err := gits.GitCreatePushURL(gitURL, details.User)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "remote", "add", "upstream", forkEnvGitURL)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "remote", "add", "origin", pushGitURL)
		if err != nil {
			return err
		}
		err = gits.GitCmd(dir, "push", "-u", "origin", "master")
		if err != nil {
			return err
		}
		*/
	}
	err = gits.GitCmd(dir, "branch", branchName)
	if err != nil {
		return err
	}
	err = gits.GitCmd(dir, "checkout", branchName)
	if err != nil {
		return err
	}

	requirementsFile := filepath.Join(dir, helm.RequirementsFileName)
	requirements, err := helm.LoadRequirementsFile(requirementsFile)
	if err != nil {
		return err
	}
	requirements.SetAppVersion(app, version)
	err = helm.SaveRequirementsFile(requirementsFile, requirements)

	err = gits.GitCmd(dir, "add", helm.RequirementsFileName)
	if err != nil {
		return err
	}
	message := fmt.Sprintf("Promote %s to version %s", app, versionName)
	err = gits.GitCommit(dir, message)
	if err != nil {
		return err
	}
	err = gits.GitPush(dir)
	if err != nil {
		return err
	}

	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
	  return err
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
	  return err
	}

	provider, err := gitInfo.PickOrCreateProvider(authConfigSvc)
	if err != nil {
	  return err
	}

	gha := &gits.GitPullRequestArguments{
		Owner: gitInfo.Organisation,
		Repo:  gitInfo.Name,
		Title: app + " to " + versionName,
		Body:  message,
		Base:  base,
		Head:  branchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
	  return err
	}
	o.Printf("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return nil
}

func (o *PromoteOptions) GetTargetNamespace(ns string, env string) (string, *v1.Environment, error) {
	kubeClient, currentNs, err := o.KubeClient()
	if err != nil {
		return "", nil, err
	}
	team, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return "", nil, err
	}

	jxClient, _, err := o.JXClient()
	if err != nil {
		return "", nil, err
	}

	var envResource *v1.Environment
	targetNS := currentNs
	if env != "" {
		m, envNames, err := kube.GetEnvironments(jxClient, team)
		if err != nil {
			return "", nil, err
		}
		if len(envNames) == 0 {
			return "", nil, fmt.Errorf("No Environments have been created yet in team %s. Please create some via 'jx create env'", team)
		}
		envResource = m[env]
		if envResource == nil {
			return "", nil, util.InvalidOption(optionEnvironment, env, envNames)
		}
		targetNS = envResource.Spec.Namespace
		if targetNS == "" {
			return "", nil, fmt.Errorf("Environment %s does not have a namspace associated with it!", env)
		}
	} else if ns != "" {
		targetNS = ns
	}

	labels := map[string]string{}
	annotations := map[string]string{}
	err = kube.EnsureNamespaceCreated(kubeClient, targetNS, labels, annotations)
	if err != nil {
		return "", nil, err
	}
	return targetNS, envResource, nil
}
