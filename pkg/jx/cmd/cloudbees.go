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

type CloudBeesOptions struct {
	CommonOptions

	OnlyViewURL bool
}

var (
	// TODO this won't work yet as the ingress can't handle fake paths
	appendTeam = false

	cdx_long = templates.LongDesc(`
		Opens the CloudBees app for Kubernetes in a browser.

		Which helps you visualise your CI/CD pipelines, apps, environments and teams.
`)
	cdx_example = templates.Examples(`
		# Open the CDX dashboard in a browser
		jx cloudbees

		# Print the Jenkins X console URL but do not open a browser
		jx console -u`)
)

func NewCmdCloudBees(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CloudBeesOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "cloudbees",
		Short:   "Opens the CloudBees app for Kubernetes for visualising CI/CD and your environments",
		Long:    cdx_long,
		Example: cdx_example,
		Aliases: []string{"cloudbee", "cb", "cdx"},
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

func (o *CloudBeesOptions) Run() error {
	client, ns, err := o.KubeClient()
	if err != nil {
		return err
	}

	url, err := kube.GetServiceURLFromName(client, kube.ServiceCloudBees, defaultCloudBeesNamespace)
	if err != nil {
		return err
	}

	if appendTeam {
		devNs, _, err := kube.GetDevNamespace(client, ns)
		if err != nil {
			return err
		}
		if devNs != "" {
			url = util.UrlJoin(url, "teams", devNs)
		}
	}
	return o.OpenURL(url, "CloudBees")
}

func (o *CloudBeesOptions) Open(name string, label string) error {
	url, err := o.findService(name)
	if err != nil {
		return err
	}
	return o.OpenURL(url, label)
}

func (o *CloudBeesOptions) OpenURL(url string, label string) error {
	fmt.Fprintf(o.Out, "%s: %s\n", label, util.ColorInfo(url))
	if !o.OnlyViewURL {
		browser.OpenURL(url)
	}
	return nil
}
