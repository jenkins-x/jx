package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fatih/color"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
)

const (
	optionLabelColor     = "label-color"
	optionNamespaceColor = "namespace-color"
	optionContextColor   = "context-color"
)

// PromptOptions containers the CLI options
type PromptOptions struct {
	CommonOptions

	NoLabel  bool
	ShowIcon bool

	Prefix    string
	Label     string
	Separator string
	Divider   string
	Suffix    string

	LabelColor     []string
	NamespaceColor []string
	ContextColor   []string
}

var (
	get_prompt_long = templates.LongDesc(`
		Generate a command prompt for the current namespace and kubernetes context.
`)

	get_prompt_example = templates.Examples(`
		# Generate the current prompt
		jx prompt

		# Enable the prompt for bash
		PS1="[\u@\h \W \$(jx prompt)]\$ "

		# Enable the prompt for zsh
		PROMPT='$(jx prompt)'$PROMPT
	`)
)

// NewCmdPrompt creates the new command for: jx get prompt
func NewCmdPrompt(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &PromptOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "prompt",
		Short:   "Generate the command line prompt for the current team and environment",
		Long:    get_prompt_long,
		Example: get_prompt_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Prefix, "prefix", "p", "", "The prefix text for the prompt")
	cmd.Flags().StringVarP(&options.Label, "label", "l", "k8s", "The label for the prompt")
	cmd.Flags().StringVarP(&options.Separator, "separator", "s", ":", "The separator between the label and the rest of the prompt")
	cmd.Flags().StringVarP(&options.Divider, "divider", "d", ":", "The divider between the team and environment for the prompt")
	cmd.Flags().StringVarP(&options.Suffix, "suffix", "x", ">", "The suffix text for the prompt")

	cmd.Flags().StringArrayVarP(&options.LabelColor, optionLabelColor, "", []string{"blue"}, "The color for the label")
	cmd.Flags().StringArrayVarP(&options.NamespaceColor, optionNamespaceColor, "", []string{"green"}, "The color for the namespace")
	cmd.Flags().StringArrayVarP(&options.ContextColor, optionContextColor, "", []string{"cyan"}, "The color for the kubernetes context")

	cmd.Flags().BoolVarP(&options.NoLabel, "no-label", "", false, "Disables the use of the label in the prompt")
	cmd.Flags().BoolVarP(&options.ShowIcon, "icon", "i", false, "Uses an icon for the label in the prompt")

	return cmd
}

// Run implements this command
func (o *PromptOptions) Run() error {
	config, _, err := kube.LoadConfig()

	context := config.CurrentContext
	namespace := kube.CurrentNamespace(config)

	// enable color
	color.NoColor = os.Getenv("TERM") == "dumb"

	label := o.Label
	separator := o.Separator
	divider := o.Divider
	prefix := o.Prefix
	suffix := o.Suffix

	labelColor, err := cmdutil.GetColor(optionLabelColor, o.LabelColor)
	if err != nil {
		return err
	}
	nsColor, err := cmdutil.GetColor(optionLabelColor, o.NamespaceColor)
	if err != nil {
		return err
	}
	ctxColor, err := cmdutil.GetColor(optionLabelColor, o.ContextColor)
	if err != nil {
		return err
	}
	if o.NoLabel {
		label = ""
		separator = ""
	} else {
		if o.ShowIcon {
			label = "☸️  "
			label = labelColor.Sprint(label)
		} else {
			label = labelColor.Sprint(label)
		}
	}
	if namespace == "" {
		divider = ""
	} else {
		namespace = nsColor.Sprint(namespace)
	}
	context = ctxColor.Sprint(context)
	log.Infof("%s\n", strings.Join([]string{prefix, label, separator, namespace, divider, context, suffix}, ""))
	return nil
}
