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

type CDXOptions struct {
	CommonOptions

	OnlyViewURL bool
}

var (
	// TODO this won't work yet as the ingress can't handle fake paths
	appendTeam = false

	cdx_long = templates.LongDesc(`
		Opens the CDX dashboard in a browser.

		Which helps you visualise your CI / CD pipelines, apps, environments and teams.
`)
	cdx_example = templates.Examples(`
		# Open the CDX dashboard in a browser
		jx cdx

		# Print the Jenkins X console URL but do not open a browser
		jx console -u`)
)

func NewCmdCDX(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CDXOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "cdx",
		Short:   "Opens the CDX dashboard for visualising CI / CD and your environments",
		Long:    cdx_long,
		Example: cdx_example,
		Aliases: []string{"dashboard"},
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

func (o *CDXOptions) Run() error {
	url, err := o.findService(kube.ServiceCDX)
	if err != nil {
		o.warnf("It looks like you are not running the CDX addon.\nDid you try running this command: 'jx create addon cdx'\n")
		return err
	}
	if appendTeam {
		f := o.Factory
		client, ns, err := f.CreateClient()
		if err != nil {
			return err
		}
		devNs, _, err := kube.GetDevNamespace(client, ns)
		if err != nil {
			return err
		}
		if devNs != "" {
			url = util.UrlJoin(url, "teams", devNs)
		}
	}
	return o.OpenURL(url, "CDX")
}

func (o *CDXOptions) Open(name string, label string) error {
	url, err := o.findService(name)
	if err != nil {
		return err
	}
	return o.OpenURL(url, label)
}

func (o *CDXOptions) OpenURL(url string, label string) error {
	fmt.Fprintf(o.Out, "%s: %s\n", label, util.ColorInfo(url))
	if !o.OnlyViewURL {
		browser.OpenURL(url)
	}
	return nil
}
