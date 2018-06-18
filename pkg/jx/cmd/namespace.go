package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
	"k8s.io/client-go/kubernetes"
	"sort"
)

type NamespaceOptions struct {
	CommonOptions
}

const (
	noContextDefinedError = "There is no context defined in your kubernetes configuration"
)

var (
	namespace_long = templates.LongDesc(`
		Displays or changes the current namespace.`)
	namespace_example = templates.Examples(`
		# view the current namespace
		jx ns -b

		# to select the namespace to switch to
		jx ns

		# Change the current namespace to 'cheese'
		jx ns cheese`)
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
		Short:   "View or change the current namespace context in the current kubernetes cluster",
		Long:    namespace_long,
		Example: namespace_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)
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
	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	names, err := GetNamespaceNames(client)
	if err != nil {
		return err
	}
	currentNS := kube.CurrentNamespace(config)

	if ns == "" && !o.BatchMode {
		defaultNamespace := ""
		ctx := kube.CurrentContext(config)
		if ctx != nil {
			defaultNamespace = currentNS
		}
		pick, err := o.PickNamespace(names, defaultNamespace)
		if err != nil {
			return err
		}
		ns = pick
	}
	info := util.ColorInfo
	if ns != "" && ns != currentNS {
		_, err = client.CoreV1().Namespaces().Get(ns, meta_v1.GetOptions{})
		if err != nil {
			return util.InvalidArg(ns, names)
		}
		newConfig := *config
		ctx := kube.CurrentContext(config)
		if ctx == nil {
			return fmt.Errorf(noContextDefinedError)
		}
		if ctx.Namespace == ns {
			return nil
		}
		ctx.Namespace = ns
		err = clientcmd.ModifyConfig(po, newConfig, false)
		if err != nil {
			return fmt.Errorf("Failed to update the kube config %s", err)
		}
		fmt.Fprintf(o.Out, "Now using namespace '%s' on server '%s'.\n", info(ctx.Namespace), info(kube.Server(config, ctx)))
	} else {
		ns := kube.CurrentNamespace(config)
		server := kube.CurrentServer(config)
		fmt.Fprintf(o.Out, "Using namespace '%s' from context named '%s' on server '%s'.\n", info(ns), info(config.CurrentContext), info(server))
	}
	return nil
}

// GetNamespaceNames returns the sorted list of environment names
func GetNamespaceNames(client kubernetes.Interface) ([]string, error) {
	names := []string{}
	list, err := client.CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("Failed to load Namespaces %s", err)
	}
	for _, n := range list.Items {
		names = append(names, n.Name)
	}
	sort.Strings(names)
	return names, nil
}

func (o *NamespaceOptions) PickNamespace(names []string, defaultNamespace string) (string, error) {
	if len(names) == 0 {
		return "", nil
	}
	if len(names) == 1 {
		return names[0], nil
	}
	name := ""
	prompt := &survey.Select{
		Message: "Change namespace:",
		Options: names,
		Default: defaultNamespace,
	}
	err := survey.AskOne(prompt, &name, nil)
	return name, err
}
