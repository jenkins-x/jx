package dashboard

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/services"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"

	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/pkg/browser"
	"k8s.io/client-go/kubernetes"
)

type Options struct {
	options.BaseOptions
	KubeClient  kubernetes.Interface
	Namespace   string
	ServiceName string
	NoBrowser   bool
	Quiet       bool
}

var (
	cmdLong = templates.LongDesc(`
		View the Jenkins X Pipelines Dashboard.`)

	cmdExample = templates.Examples(`
		# open the dashboard
		jx dashboard

		# display the URL only without opening a browser
		jx --no-open

		# change the current namespace to 'cheese'
		jx ns cheese

		# change the current namespace to 'brie' creating it if necessary
	    jx ns --create brie

		# switch to the namespace of the staging environment
		jx ns --env staging

		# switch back to the dev environment namespace
		jx ns --e dev

		# interactively select the Environment to switch to
		jx ns --pick
`)

	info = termcolor.ColorInfo
)

// NewCmdDashboard opens the dashboard
func NewCmdDashboard() (*cobra.Command, *Options) {
	o := &Options{}
	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"dash", "viualiser", "viualizer"},
		Short:   "View the Jenkins X Pipelines Dashboard",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&o.NoBrowser, "no-open", "", false, "Disable opening the URL; just show it on the console")
	cmd.Flags().StringVarP(&o.ServiceName, "name", "n", "jx-pipelines-visualizer", "The name of the dashboard service")
	o.BaseOptions.AddBaseFlags(cmd)
	return cmd, o
}

func (o *Options) Run() error {
	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrap(err, "creating kubernetes client")
	}
	client := o.KubeClient

	u, err := services.FindServiceURL(client, o.Namespace, o.ServiceName)
	if err != nil {
		return errors.Wrapf(err, "failed to find dashboard URL. Check you have 'chart: jx3/jx-pipelines-visualizer' in your helmfile.yaml")
	}
	if u == "" {
		return errors.Errorf("no dashboard URL. Check you have 'chart: jx3/jx-pipelines-visualizer' in your helmfile.yaml")
	}

	log.Logger().Infof("Jenkins X dashboard is running at: %s", info(u))

	if o.NoBrowser {
		return nil
	}

	err = browser.OpenURL(u)
	if err != nil {
		return err
	}
	return nil
}
