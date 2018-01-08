package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/browser"

)

type ConsoleOptions struct {
	CommonOptions

	OnlyViewURL bool
}

func NewCmdConsole(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ConsoleOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:   "console",
		Short: "Opens the Jenkins console",
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
	f := o.Factory
	client, ns, err := f.CreateClient()
	if err != nil {
		return err
	}
	name := kube.ServiceJenkins
	url, err := kube.FindServiceURL(client, ns, name)
	if url == "" {
		return fmt.Errorf("Could not find service %s in namespace %s", name, ns)
	}
	fmt.Fprintf(o.Out, "Jenkins Console: %s\n", url)
	if !o.OnlyViewURL {
		browser.OpenURL(url)
	}
	return nil
}
