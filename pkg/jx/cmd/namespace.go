package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"k8s.io/client-go/tools/clientcmd"
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
	cmd.Flags().BoolVarP(&options.Choose, "namespace", "n", false, "Choose which namespace to switch to")
	return cmd
}

func (o *NamespaceOptions) Run() error {
	config, po, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	args := o.Args
	if len(args) > 0 {
		newConfig := *config
		ctx := kube.CurrentContext(config)
		if ctx == nil {
			return fmt.Errorf(noContextDefinedError)
		}
		ctx.Namespace = args[0]
		err = clientcmd.ModifyConfig(po, newConfig, false)
		if err != nil {
			return fmt.Errorf("Failed to update the kube config %s", err)
		}
	} else {
		ctx := kube.CurrentContext(config)
		if ctx == nil {
			return fmt.Errorf(noContextDefinedError)
		}
		fmt.Fprintf(o.Out, "Using namespace '%s' from context named '%s' on server '%s'.\n", ctx.Namespace, config.CurrentContext, kube.Server(config, ctx))
	}
	return nil
}
