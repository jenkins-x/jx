package cmd

import (
	"io"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
)

var (
	create_env_long = templates.LongDesc(`
		Creates a new Environment

		An Environment maps to a kubernetes cluster and namespace and is a place that your team's applications can be promoted to via Continous Delivery.

		You can optionally use GitOps to manage the configuration of an Environment by storing all configuration in a git repository and then only changing it via Pull Requests and CI / CD.
`)

	create_env_example = templates.Examples(`
		# Create a new Environment, prompting for the required data
		jx create env

		# Creates a new Environment passing in the required data on the command line
		jx create env -n prod -l Production --no-gitops --namespace my-prod
	`)
)

// CreateEnvOptions the options for the create spring command
type CreateEnvOptions struct {
	CreateOptions

	Options           kube.Environment
	PromotionStrategy string

	NoGitOps bool
}

// NewCmdCreateEnv creates a command object for the "create" command
func NewCmdCreateEnv(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateEnvOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "environment",
		Short:   "Create a new Environment which is used to promote your Team's Applications via Continuous Delivery",
		Aliases: []string{"env"},
		Long:    create_env_long,
		Example: create_env_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	addCreateFlags(cmd, &options.CreateOptions)

	cmd.Flags().StringVarP(&options.Options.Name, "name", "n", "", "The Environment resource name. Must follow the kubernetes name conventions like Services, Namespaces")
	cmd.Flags().StringVarP(&options.Options.Spec.Label, "label", "l", "", "The Environment label which is a descriptive string like 'Production' or 'Staging'")
	cmd.Flags().StringVarP(&options.Options.Spec.Namespace, "namespace", "s", "", "The Kubernetes namespace for the Environment")
	cmd.Flags().StringVarP(&options.Options.Spec.Cluster, "cluster", "c", "", "The Kubernetes cluster for the Environment. If blank and a namespace is specified assumes the current cluster")
	cmd.Flags().StringVarP(&options.PromotionStrategy, "promotion", "p", string(kube.PromotionStrategyTypeAutomatic), "The promotion strategy")

	cmd.Flags().BoolVarP(&options.NoGitOps, "no-gitops", "x", false, "Disables the use of GitOps on the environment so that promotion is implemented by directly modifying the resources via helm instead of using a git repository")
	return cmd
}

// Run implements the command
func (o *CreateEnvOptions) Run() error {
	_, ns, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}

	env := kube.Environment{}
	o.Options.Spec.PromotionStrategy = kube.PromotionStrategyType(o.PromotionStrategy)
	err = kube.CreateEnvironmentSurvey(&env, &o.Options, o.NoGitOps, ns)
	if err != nil {
		return err
	}
	o.Printf("Created environment %#v\n", env)
	return nil
}
