package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
	"k8s.io/client-go/tools/clientcmd"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
)

type ContextOptions struct {
	CommonOptions
}

var (
	context_long = templates.LongDesc(`
		Displays or changes the current kubernetes context (cluster).`)
	context_example = templates.Examples(`
		# to select the context to switch to
		jx context

		# or the more concise alias
		jx ctx

		# view the current context
		jx ctx -b

		# Change the current namespace to 'minikube'
		jx ctx minikube`)
)

func NewCmdContext(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ContextOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "context",
		Aliases: []string{"ctx"},
		Short:   "View or change the current kubernetes context (kubernetes cluster)",
		Long:    context_long,
		Example: context_example,
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

func (o *ContextOptions) Run() error {
	config, po, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	if config == nil || config.Contexts == nil || len(config.Contexts) == 0 {
		return fmt.Errorf("No kubernetes contexts available! Try create or connect to cluster?")
	}

	contextNames := []string{}
	for k, v := range config.Contexts {
		if k != "" && v != nil {
			contextNames = append(contextNames, k)
		}
	}
	sort.Strings(contextNames)

	ctxName := ""
	args := o.Args
	if len(args) > 0 {
		ctxName = args[0]
		if util.StringArrayIndex(contextNames, ctxName) < 0 {
			return util.InvalidArg(ctxName, contextNames)
		}
	}

	if ctxName == "" && !o.BatchMode {
		defaultCtxName := config.CurrentContext
		pick, err := o.PickContext(contextNames, defaultCtxName)
		if err != nil {
			return err
		}
		ctxName = pick
	}
	info := util.ColorInfo
	if ctxName != "" && ctxName != config.CurrentContext {
		ctx := config.Contexts[ctxName]
		if ctx == nil {
			return fmt.Errorf("Could not find kubernetes context %s", ctxName)
		}
		newConfig := *config
		newConfig.CurrentContext = ctxName
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

func (o *ContextOptions) PickContext(names []string, defaultValue string) (string, error) {
	if len(names) == 0 {
		return "", nil
	}
	if len(names) == 1 {
		return names[0], nil
	}
	name := ""
	prompt := &survey.Select{
		Message: "Change kubernetes context:",
		Options: names,
		Default: defaultValue,
	}
	err := survey.AskOne(prompt, &name, nil)
	return name, err
}
