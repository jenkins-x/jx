package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/browser"
)

type ConsoleOptions struct {
	CommonOptions

	OnlyViewURL bool
}

var (
	console_long = templates.LongDesc(`
		Opens the Jenkins X console in a browser.`)
	console_example = templates.Examples(`
		# Open the Jenkins X console in a browser
		jx console

		# Print the Jenkins X console URL but do not open a browser
		jx console -u`)
)

func NewCmdConsole(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ConsoleOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "console",
		Short:   "Opens the Jenkins console",
		Long:    console_long,
		Example: console_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.OnlyViewURL, "url", "u", false, "Only displays and the URL and does not open the browser")
	return cmd
}

func (o *ConsoleOptions) Run() error {
	return o.Open(kube.ServiceJenkins, "Jenkins Console")
}

func (o *ConsoleOptions) Open(name string, label string) error {
	f := o.Factory
	client, ns, err := f.CreateClient()
	if err != nil {
		return err
	}
	devNs, _, err := kube.GetDevNamespace(client, ns)
	if err != nil {
		return err
	}
	url, err := kube.FindServiceURL(client, ns, name)
	if url == "" {
		url, err = kube.FindServiceURL(client, devNs, name)
	}
	if url == "" {
		names, err := kube.GetServiceNames(client, ns, name)
		if err != nil {
			return err
		}
		if len(names) > 1 {
			name, err = util.PickName(names, "Pick service to open: ")
			if err != nil {
				return err
			}
			if name != "" {
				url, err = kube.FindServiceURL(client, ns, name)
			}
		} else if len(names) == 1 {
			// must have been a filter
			url, err = kube.FindServiceURL(client, ns, names[0])
		}
		if url == "" {
			return fmt.Errorf("Could not find URL for service %s in namespace %s", name, ns)
		}
	}
	fmt.Fprintf(o.Out, "%s: %s\n", label, util.ColorInfo(url))
	if !o.OnlyViewURL {
		browser.OpenURL(url)
	}
	return nil
}
