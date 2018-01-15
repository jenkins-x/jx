package cmd

import (
	"github.com/spf13/cobra"
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	edit_env_long = templates.LongDesc(`
		Edits a new Environment
        ` + env_description + `
`)

	edit_env_example = templates.Examples(`
		# Edit the stating Environment, prompting for the required data
		jx edit env -n stating

		# Edit the prod Environment in batch mode (so not interactive)
		jx edit env -b -n prod -l Production --no-gitops --namespace my-prod
	`)
)

// EditEnvOptions the options for the create spring command
type EditEnvOptions struct {
	CreateOptions

	Options           v1.Environment
	PromotionStrategy string

	NoGitOps bool
}

// NewCmdEditEnv creates a command object for the "create" command
func NewCmdEditEnv(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &EditEnvOptions{
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
		Long:    edit_env_long,
		Example: edit_env_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	addCreateFlags(cmd, &options.CreateOptions)

	cmd.Flags().StringVarP(&options.Options.Name, kube.OptionName, "n", "", "The Environment resource name. Must follow the kubernetes name conventions like Services, Namespaces")
	cmd.Flags().StringVarP(&options.Options.Spec.Label, "label", "l", "", "The Environment label which is a descriptive string like 'Production' or 'Staging'")
	cmd.Flags().StringVarP(&options.Options.Spec.Namespace, kube.OptionNamespace, "s", "", "The Kubernetes namespace for the Environment")
	cmd.Flags().StringVarP(&options.Options.Spec.Cluster, "cluster", "c", "", "The Kubernetes cluster for the Environment. If blank and a namespace is specified assumes the current cluster")
	cmd.Flags().StringVarP(&options.PromotionStrategy, "promotion", "p", "", "The promotion strategy")

	cmd.Flags().BoolVarP(&options.NoGitOps, "no-gitops", "x", false, "Disables the use of GitOps on the environment so that promotion is implemented by directly modifying the resources via helm instead of using a git repository")
	return cmd
}

// Run implements the command
func (o *EditEnvOptions) Run() error {
	jxClient, currentNs, err := o.Factory.CreateJXClient()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	kube.RegisterEnvironmentCRD(apisClient)

	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}

	envNames, err := kube.GetEnvironmentNames(jxClient, ns)
	if err != nil {
		return err
	}
	name := ""
	args := o.Args
	if len(args) > 0 {
		name = args[0]
	} else {
		name = o.Options.Name
		if name == "" {
			name, err = kube.PickEnvironment(envNames)
			if err != nil {
				return err
			}
		}
	}

	env, err := jxClient.JenkinsV1().Environments(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return util.InvalidArg(name, envNames)
	}

	o.Options.Spec.PromotionStrategy = v1.PromotionStrategyType(o.PromotionStrategy)
	err = kube.CreateEnvironmentSurvey(env, &o.Options, o.NoGitOps, ns, jxClient)
	if err != nil {
		return err
	}
	_, err = jxClient.JenkinsV1().Environments(ns).Update(env)
	if err != nil {
		return err
	}
	o.Printf("Updated environment %s\n", util.ColorInfo(env.Name))

	return kube.EnsureEnvironmentNamespaceSetup(kubeClient, jxClient, env, ns)
}
