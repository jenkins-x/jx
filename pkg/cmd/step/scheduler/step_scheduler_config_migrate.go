package scheduler

import (
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StepSchedulerConfigMigrateOptions contains the command line flags
type StepSchedulerConfigMigrateOptions struct {
	opts.StepOptions
	Agent                   string
	ProwConfigFileLocation  string
	ProwPluginsFileLocation string
	SkipVerification        bool
	DryRun                  bool
	// allow git to be configured externally before a PR is created
	ConfigureGitCallback gits.ConfigureGitFn
}

var (
	stepSchedulerConfigMigrateLong = templates.LongDesc(`
        This command will transform your pipeline schedulers in to prow config. 
        If you are using gitops the prow config will be added to your environment repository. 
        For non-gitops environments the prow config maps will applied to your dev environment.
`)
	stepSchedulerConfigMigrateExample = templates.Examples(`
	
	jx step scheduler config migrate
`)
)

// NewCmdStepSchedulerConfigMigrate Steps a command object for the "step" command
func NewCmdStepSchedulerConfigMigrate(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepSchedulerConfigMigrateOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "migrate",
		Short:   "scheduler config migrate",
		Long:    stepSchedulerConfigMigrateLong,
		Example: stepSchedulerConfigMigrateExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	options.AddCommonFlags(cmd)
	cmd.Flags().StringVarP(&options.Agent, "agent", "", "prow", "The scheduler agent to use e.g. Prow")
	cmd.Flags().StringVarP(&options.ProwConfigFileLocation, "prow-config-file", "", "", "The location of the config file to use")
	cmd.Flags().StringVarP(&options.ProwPluginsFileLocation, "prow-plugins-file", "", "", "The location of the plugins file to use")
	cmd.Flags().BoolVarP(&options.SkipVerification, "skipVerification", "", false, "Skip verification of the new configuration")
	cmd.Flags().BoolVarP(&options.DryRun, "dryRun", "", true, "Do not apply the generated configuration")
	return cmd
}

// Run implements this command
func (o *StepSchedulerConfigMigrateOptions) Run() error {
	gitOps, devEnv := o.GetDevEnv()
	jxClient, ns, err := o.JXClient()
	if err != nil {
		return errors.WithStack(err)
	}
	teamSettings, err := o.TeamSettings()
	if err != nil {
		return errors.WithStack(err)
	}
	switch o.Agent {
	case "prow":
		kubeClient, err := o.KubeClient()
		if err != nil {
			return errors.WithStack(err)
		}
		sourceRepoGroups, sourceRepos, schedulers, err := pipelinescheduler.CreateSchedulersFromProwConfig(o.ProwConfigFileLocation, o.ProwPluginsFileLocation, o.SkipVerification, o.DryRun, gitOps, jxClient, kubeClient, ns, teamSettings.DefaultScheduler.Name, devEnv)
		if err != nil {
			return errors.Wrapf(err, "generating Prow config")
		}
		if !o.DryRun {
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

				gitProvider, _, err := o.CreateGitProviderForURLWithoutKind(devEnv.Spec.Source.URL)
				if err != nil {
					return errors.Wrapf(err, "creating git provider for %s", devEnv.Spec.Source.URL)
				}
				opts.GitProvider = gitProvider
				opts.ConfigureGitFn = o.ConfigureGitCallback
				opts.Gitter = o.Git()
				opts.Helmer = o.Helm()
				err = opts.AddSchedulersToEnvironmentRepo(sourceRepoGroups, sourceRepos, schedulers)
				if err != nil {
					return errors.Wrapf(err, "adding pipeline scheduler config to environment repo")
				}
			} else {
				err = pipelinescheduler.ApplySchedulersDirectly(jxClient, ns, sourceRepoGroups, sourceRepos, schedulers, devEnv)
				if err != nil {
					return errors.Wrapf(err, "applying pipeline scheduler config")
				}
			}
		} else {
			log.Logger().Info("Running in dry run mode, pipeline scheduler config will be discarded")

		}
	default:
		return errors.Errorf("%s is an unsupported agent. Available agents are: prow", o.Agent)
	}
	return nil
}
