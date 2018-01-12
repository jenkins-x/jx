package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEnvOptions containers the CLI options
type GetEnvOptions struct {
	CommonOptions
}

var (
	get_env_long = templates.LongDesc(`
		Display one or many environments.
`)

	get_env_example = templates.Examples(`
		# List all environments
		jx get environments

		# List all environments using the shorter alias
		jx get env
	`)
)

// NewCmdGetEnv creates the new command for: jx get env
func NewCmdGetEnv(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetEnvOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "environment",
		Short:   "Display one or many Enviroments",
		Aliases: []string{"env"},
		Long:    get_env_long,
		Example: get_env_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	return cmd
}

// Run implements this command
func (o *GetEnvOptions) Run() error {
	f := o.Factory
	client, currentNs, err := f.CreateJXClient()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}
	envs, err := client.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(envs.Items) == 0 {
		return outputEmptyListWarning(o.Out)
	}

	table := o.CreateTable()
	table.AddRow("NAME", "LABEL", "PROMOTE", "NAMESPACE", "CLUSTER", "SOURCE URL", "REF")

	for _, env := range envs.Items {
		table.AddRow(env.Name, env.Spec.Label, string(env.Spec.PromotionStrategy), env.Spec.Namespace, env.Spec.Cluster, env.Spec.Source.URL, env.Spec.Source.Ref)
	}
	table.Render()
	return nil
}
