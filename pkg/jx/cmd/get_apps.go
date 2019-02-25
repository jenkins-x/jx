package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

// GetAppsOptions containers the CLI options
type GetAppsOptions struct {
	GetOptions
	Namespace string
	GitOps    bool
	DevEnv    *v1.Environment
}

var (
	getAppsLong = templates.LongDesc(`
		Display installed Apps
`)

	getAppsExample = templates.Examples(`
		# List all apps
		jx get apps

		# Display details about the app called cheese
		jx get app cheese

	`)
)

// NewCmdGetApps creates the new command for: jx get version
func NewCmdGetApps(commonOpts *CommonOptions) *cobra.Command {
	options := &GetAppsOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "Apps",
		Short:   "Display one or more installed Apps",
		Aliases: []string{"app", "apps"},
		Long:    getAppsLong,
		Example: getAppsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetAppsOptions) Run() error {
	kubeClient, err := o.KubeClient()
	o.GitOps, o.DevEnv = o.GetDevEnv()
	if err != nil {
		return err
	}
	jxClient, _, err := o.JXClientAndDevNamespace()
	if err != nil {
		return errors.Wrapf(err, "getting jx client")
	}
	opts := apps.InstallOptions{
		In:        o.In,
		DevEnv:    o.DevEnv,
		Verbose:   o.Verbose,
		Err:       o.Err,
		Out:       o.Out,
		GitOps:    o.GitOps,
		BatchMode: o.BatchMode,
		Helmer:    o.Helm(),
		JxClient:  jxClient,
	}

	apps, err := opts.GetApps(kubeClient, o.Namespace, o.Args)
	if err != nil {
		return err
	}

	if len(apps.Items) == 0 {
		fmt.Fprint(o.Out, "No Apps found\n")
		return nil
	}

	table := o.generateTable(apps, kubeClient)

	table.Render()
	return nil
}

func (o *GetAppsOptions) generateTable(apps *v1.AppList, kubeClient kubernetes.Interface) table.Table {
	table := o.generateTableHeaders(apps)
	for _, app := range apps.Items {
		name := app.Labels[helm.LabelAppName]
		if name != "" {
			version := app.Labels[helm.LabelAppVersion]
			description := app.Annotations[helm.AnnotationAppDescription]
			repository := app.Annotations[helm.AnnotationAppRepository]
			row := []string{name, version, description, repository}
			table.AddRow(row...)
		}
	}
	return table
}

func (o *GetAppsOptions) generateTableHeaders(apps *v1.AppList) table.Table {
	t := o.createTable()
	t.Out = o.CommonOptions.Out
	titles := []string{"Name", "Version", "Description", "Chart Repository"}
	t.AddRow(titles...)
	return t
}
