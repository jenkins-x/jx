package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"k8s.io/client-go/tools/clientcmd"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gopkg.in/AlecAivazis/survey.v1"
	"k8s.io/client-go/kubernetes"
)

type NamespaceOptions struct {
	CommonOptions

	Choose bool
}

const (
	noContextDefinedError = "There is no context defined in your kubernetes configuration"
)

var (
	namespace_long = templates.LongDesc(`
		Displays or changes the current namespace.`)
	namespace_example = templates.Examples(`
		# view the current namespace
		jx namespace

		# view the current namespace (concise version)
		jx ns

		# Change the current namespace to 'cheese'
		jx ns cheese

		# Select which namespace to change to from the available namespaces
		jx ns -s`)
)

func NewCmdNamespace(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &NamespaceOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "namespace",
		Aliases: []string{"ns"},
		Short:   "View or change the current namespace context in the current kubernetes clsuter",
		Long:    namespace_long,
		Example: namespace_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Choose, "select", "s", false, "Select which namspace to switch to")
	return cmd
}

func (o *NamespaceOptions) Run() error {
	config, po, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	ns := ""
	args := o.Args
	if len(args) > 0 {
		ns = args[0]
	}
	client, _, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	if o.Choose {
		defaultNamespace := ""
		ctx := kube.CurrentContext(config)
		if ctx != nil {
			defaultNamespace = kube.CurrentNamespace(config)
		}
		pick, err := o.PickNamespace(client, defaultNamespace)
		if err != nil {
			return err
		}
		ns = pick
	}
	if ns != "" {
		_, err = client.CoreV1().Namespaces().Get(ns, meta_v1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Error: %s\nIf you want to create that namespace then try:\n    kubectl create ns %s", err, ns)
		}
		newConfig := *config
		ctx := kube.CurrentContext(config)
		if ctx == nil {
			return fmt.Errorf(noContextDefinedError)
		}
		ctx.Namespace = ns
		err = clientcmd.ModifyConfig(po, newConfig, false)
		if err != nil {
			return fmt.Errorf("Failed to update the kube config %s", err)
		}
		fmt.Fprintf(o.Out, "Now using namespace '%s' on server '%s'.\n", ctx.Namespace, kube.Server(config, ctx))
	} else {
		ctx := kube.CurrentContext(config)
		if ctx == nil {
			return fmt.Errorf(noContextDefinedError)
		}
		fmt.Fprintf(o.Out, "Using namespace '%s' from context named '%s' on server '%s'.\n", ctx.Namespace, config.CurrentContext, kube.Server(config, ctx))
	}
	return nil
}

func (o *NamespaceOptions) PickNamespace(client *kubernetes.Clientset, defaultNamespace string) (string, error) {
	list, err := client.CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("Failed to load current namespaces %s", err)
	}
	names := []string{}
	for _, n := range list.Items {
		names = append(names, n.Name)
	}
	var qs = []*survey.Question{
		{
			Name: "namespace",
			Prompt: &survey.Select{
				Message: "Change namespace: ",
				Options: names,
				Default: defaultNamespace,
			},
		},
	}
	answers := struct {
		Namespace string
	}{}
	err = survey.Ask(qs, &answers)
	if err != nil {
		return "", err
	}
	return answers.Namespace, nil
}
