package cmd

import (
	"github.com/jenkins-x/jx/pkg/environments"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StepSchedulerConfigApplyOptions contains the command line flags
type StepSchedulerConfigApplyOptions struct {
	StepOptions
	Agent string

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback environments.ConfigureGitFn
}

const (
	optionOrg = "org"
)

var (
	stepSchedulerConfigApplyLong = templates.LongDesc(`
		This pipeline step command allows you to generate the scheduler agent configuration from the Jenkins X pipeline
scheduler configuration.

`)
	stepSchedulerConfigApplyExample = templates.Examples(`
	
	jx step scheduler config generate
`)
)

// NewCmdStepSchedulerConfigApply Steps a command object for the "step" command
func NewCmdStepSchedulerConfigApply(commonOpts *CommonOptions) *cobra.Command {
	options := &StepSchedulerConfigApplyOptions{
		StepOptions: StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "apply",
		Short:   "scheduler config apply",
		Long:    stepSchedulerConfigApplyLong,
		Example: stepSchedulerConfigApplyExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
	cmd.Flags().StringVarP(&options.Agent, "agent", "", "prow", "The scheduler agent to use e.g. Prow")
	return cmd
}

// Run implements this command
func (o *StepSchedulerConfigApplyOptions) Run() error {
	gitOps, devEnv := o.GetDevEnv()
	switch o.Agent {
	case "prow":
		jxClient, ns, err := o.JXClient()
		if err != nil {
			return errors.WithStack(err)
		}
		cfg, plugs, err := pipelinescheduler.GenerateProw(jxClient, ns)
		if err != nil {
			return errors.Wrapf(err, "generating Prow config")
		}
		if gitOps {
			opts := pipelinescheduler.GitOpsOptions{
				Verbose: o.Verbose,
				DevEnv:  devEnv,
			}
			environmentsDir, err := o.EnvironmentsDir()
			if err != nil {
				return errors.Wrapf(err, "getting environments dir")
			}
			opts.EnvironmentsDir = environmentsDir

			gitProvider, _, err := o.createGitProviderForURLWithoutKind(devEnv.Spec.Source.URL)
			if err != nil {
				return errors.Wrapf(err, "creating git provider for %s", devEnv.Spec.Source.URL)
			}
			opts.GitProvider = gitProvider
			opts.ConfigureGitFn = o.ConfigureGitCallback
			opts.Gitter = o.Git()
			opts.Helmer = o.Helm()
			err = opts.AddToEnvironmentRepo(cfg, plugs)
			if err != nil {
				return errors.Wrapf(err, "adding Prow config to environment repo")
			}
		} else {
			kubeClient, ns, err := o.KubeClientAndNamespace()
			if err != nil {
				return errors.WithStack(err)
			}
			err = pipelinescheduler.ApplyDirectly(kubeClient, ns, cfg, plugs)
			if err != nil {
				return errors.Wrapf(err, "applying Prow config")
			}
		}
	default:
		return errors.Errorf("%s is an unsupported agent. Available agents are: prow", o.Agent)
	}
	return nil
}
