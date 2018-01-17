package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/jenkins-x/jx/pkg/kube"
)

var (
	delete_env_long = templates.LongDesc(`
		Deletes one or more environments.
`)

	delete_env_example = templates.Examples(`
		# Deletes an environment
		jx delete env staging
	`)
)

// DeleteEnvOptions the options for the create spring command
type DeleteEnvOptions struct {
	CreateOptions

	DeleteNamespace bool
}

// NewCmdDeleteEnv creates a command object for the "delete repo" command
func NewCmdDeleteEnv(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &DeleteEnvOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "env",
		Short:   "Deletes one or more environments",
		Aliases: []string{"environment"},
		Long:    delete_env_long,
		Example: delete_env_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	//addDeleteFlags(cmd, &options.CreateOptions)

	cmd.Flags().BoolVarP(&options.DeleteNamespace, "namespace", "n", false, "Delete the namespace for the Environment too?")
	return cmd
}

// Run implements the command
func (o *DeleteEnvOptions) Run() error {
	f := o.Factory
	jxClient, currentNs, err := f.CreateJXClient()
	if err != nil {
		return err
	}
	kubeClient, _, err := f.CreateClient()
	if err != nil {
		return err
	}
	apisClient, err := f.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	kube.RegisterEnvironmentCRD(apisClient)

	ns, currentEnv, err := kube.GetDevNamespace(kubeClient, currentNs)
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
		for _, arg := range args {
			if util.StringArrayIndex(envNames, arg) < 0 {
				return util.InvalidArg(arg, envNames)
			}
		}
		for _, arg := range args {
			err = o.deleteEnviroment(jxClient, ns, arg)
			if err != nil {
				return err
			}
		}
	} else {
		name, err = kube.PickEnvironment(envNames, currentEnv)
		if err != nil {
			return err
		}
		err = o.deleteEnviroment(jxClient, ns, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *DeleteEnvOptions) deleteEnviroment(jxClient *versioned.Clientset, ns string, name string) error {
	err := jxClient.JenkinsV1().Environments(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	o.Printf("Deleted environment %s\n", util.ColorInfo(name))
	return nil
}
