package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"k8s.io/client-go/tools/clientcmd"

	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	bashEnvPrefix = "PS1="

	defaultRcFile = `
if [ -f /etc/bashrc ]; then
    source /etc/bashrc
fi
if [ -f ~/.bashrc ]; then
    source ~/.bashrc
fi
`
)

type ShellOptions struct {
	CommonOptions
}

var (
	shell_long = templates.LongDesc(`
		Create a sub shell so that changes to the kubernetes context, namespace or environment remain local to the shell.`)
	shell_example = templates.Examples(`
		# create a new shell where the context changes are local to the shell only
		jx shell

		# create a new shell using a specific named context
		jx shell prod-cluster

		# ends the current shell and returns to the previous kubernetes context
		exit
`)
)

func NewCmdShell(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ShellOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "shell",
		Aliases: []string{"sh"},
		Short:   "Create a sub shell so that changes to the kubernetes context, namespace or environment remain local to the shell",
		Long:    shell_long,
		Example: shell_example,
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

func (o *ShellOptions) Run() error {
	config, _, err := kube.LoadConfig()
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
	if ctxName == "" {
		ctxName = config.CurrentContext
	}
	newConfig := *config
	newConfig.CurrentContext = ctxName

	tmpfile, err := ioutil.TempFile("/tmp", "jx-kube-config-")
	if err != nil {
		return err
	}
	tmpFileName := tmpfile.Name()
	err = clientcmd.WriteToFile(newConfig, tmpFileName)
	if err != nil {
		return err
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	prompt := o.createNewBashPrompt(os.Getenv("PS1"))
	rcFile := defaultRcFile + "\nexport PS1=" + prompt + "\nexport KUBECONFIG=\"" + tmpFileName + "\"\n"

	tmpRCfile, err := ioutil.TempFile("/tmp", "jx-shell-rcfile-")
	if err != nil {
		return err
	}
	tmpRCfileName := tmpRCfile.Name()
	err = ioutil.WriteFile(tmpRCfileName, []byte(rcFile), util.DefaultWritePermissions)
	if err != nil {
		return err
	}

	info := util.ColorInfo
	log.Infof("Creating a new shell using the kubernetes context %s\n", info(ctxName))
	log.Infof("Bash RC file is %s\n\n", tmpRCfileName)
	log.Infof("All changes to the kuberentes context like changing environment, namespace or context will be local to this shell\n")
	log.Infof("To return to the global context use the command: exit\n\n")

	e := exec.Command(shell, "-rcfile", tmpRCfileName, "-i")
	e.Stdout = o.Out
	e.Stderr = o.Err
	e.Stdin = os.Stdin
	return e.Run()
}

func (o *ShellOptions) PickContext(names []string, defaultValue string) (string, error) {
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

func (o *ShellOptions) createNewBashPrompt(prompt string) string {
	if prompt == "" {
		return "[\\u@\\h \\W \\$(jx prompt) ]\\$ "
	}
	if strings.Contains(prompt, "jx prompt") {
		return prompt
	}
	if prompt[0] == '"' {
		return prompt[0:1] + "\\$(jx prompt) " + prompt[1:]
	}
	if prompt[0] == '\'' {
		return prompt[0:1] + "$(jx prompt) " + prompt[1:]
	}
	return "'$(jx prompt) " + prompt + "'"
}
