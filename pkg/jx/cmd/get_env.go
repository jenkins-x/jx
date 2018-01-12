package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
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
	args := o.Args
	if len(args) > 0 {
		e := args[0]
		env, err := client.JenkinsV1().Environments(ns).Get(e, metav1.GetOptions{})
		if err != nil {
			envNames, err := kube.GetEnvironmentNames(client, ns)
			if err != nil {
				return err
			}
			return util.InvalidArg(e, envNames)
		}

		// lets output one environment
		spec := &env.Spec

		table := o.CreateTable()
		table.AddRow("ENV", "LABEL", "NAMESPACE", "SOURCE", "REF")
		table.AddRow(e, spec.Label, spec.Namespace, spec.Source.URL, spec.Source.Ref)
		table.Render()
		o.Printf("\n")

		ens := env.Spec.Namespace
		if ens != "" {
			deps, err := kubeClient.AppsV1beta2().Deployments(ens).List(metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("Could not find deployments in namespace %s: %s", ens, err)
			}

			table = o.CreateTable()
			table.AddRow("APP", "VERSION", "DESIRED", "CURRENT", "UP-TO-DATE", "AVAILABLE", "AGE")
			for _, d := range deps.Items {
				replicas := ""
				if d.Spec.Replicas != nil {
					replicas = formatInt32(*d.Spec.Replicas)
				}
				table.AddRow(d.Name, kube.GetVersion(&d.ObjectMeta), replicas,
					formatInt32(d.Status.ReadyReplicas), formatInt32(d.Status.UpdatedReplicas), formatInt32(d.Status.AvailableReplicas), "")
			}
			table.Render()
		}
	} else {
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
	}
	return nil
}

func formatInt32(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}
