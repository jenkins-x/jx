package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"strings"
)

type RshOptions struct {
	CommonOptions

	Container string
	Namespace string
}

var (
	rsh_long = templates.LongDesc(`
		Opens a terminal or runs a command in a pods container

`)

	rsh_example = templates.Examples(`
		# Open a terminal in the first container of the foo deployment's latest pod
		jx rsh foo

		# Opens a terminal in the cheese container in the latest pod in the foo deployment
		jx rsh -c cheese foo
`)
)

func NewCmdRsh(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &RshOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "rsh [deploymentOrPodName]",
		Short:   "Opens a terminal in a pod or runs a command in the pod",
		Long:    rsh_long,
		Example: rsh_example,
		Aliases: []string{"log"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Container, "container", "c", "", "The name of the container to log")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to look for the Deployment. Defaults to the current namespace")
	return cmd
}

func (o *RshOptions) Run() error {
	args := o.Args

	client, curNs, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = curNs
	}

	filter := ""
	names, err := kube.GetPodNames(client, ns, "")
	if err != nil {
		return err
	}
	if len(names) == 0 {
		if filter == "" {
			return fmt.Errorf("There are no Pods")
		} else {
			return fmt.Errorf("There are no Pods matching filter: " + filter)
		}
	}
	name := ""
	if len(args) == 0 {
		n, err := util.PickName(names, "Pick Pod:")
		if err != nil {
			return err
		}
		name = n
	} else {
		name = args[0]
		if util.StringArrayIndex(names, name) < 0 {
			// lets try use the name as a filter
			filteredNames := []string{}
			for _, n := range names {
				if strings.Contains(n, name) {
					filteredNames = append(filteredNames, n)
				}
			}
			n, err := util.PickName(filteredNames, "Pick Pod:")
			if err != nil {
				return err
			}
			name = n
		}
	}

	if name == "" {
		return fmt.Errorf("No pod found for namespace %s with name %s", ns, name)
	}

	for {
		a := []string{"exec", "-it", "-n", ns}
		if o.Container != "" {
			a = append(a, "-c", o.Container)
		}
		a = append(a, name)
		if len(args) > 1 {
			a = append(a, "--")
			a = append(a, args[1:]...)
		} else {
			a = append(a, "bash")
		}
		err = o.runCommandInteractive(true, "kubectl", a...)
		if err != nil {
			return nil
		}
	}
}
