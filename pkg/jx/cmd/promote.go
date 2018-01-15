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
	if version != "" {
		return o.runCommand("helm", "install", "--namespace", targetNS, "--version", version, fullAppName)
	}
	return o.runCommand("helm", "install", "--namespace", targetNS, fullAppName)
}

func (o *PromoteOptions) PromoteViaPullRequest(env *v1.Environment) error {
	return fmt.Errorf("TODO")
}

func (o *PromoteOptions) GetTargetNamespace(ns string, env string) (string, *v1.Environment, error) {
	kubeClient, currentNs, err := o.Factory.CreateClient()
	if err != nil {
		return "", nil, err
	}
	team, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return "", nil, err
	}

	jxClient, _, err := o.Factory.CreateJXClient()
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
