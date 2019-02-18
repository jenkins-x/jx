package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/pkg/errors"
	"io"
	"k8s.io/client-go/kubernetes"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetAppsOptions containers the CLI options
type GetAppsOptions struct {
	CommonOptions
	ReleaseName string
	Results     GetAppsResults
}

// GetAppsResults contains the data result from invoking this command
type GetAppsResults struct {
	// Apps is a map indexed by the Apps name then the environment name
	Apps map[string]map[string]*AppsEnvironmentInfo
}

// AppsEnvironmentInfo contains the results of an app for an environment
type AppsEnvironmentInfo struct {
	Deployment  *v1beta1.Deployment
	Environment *v1.Environment
	Version     string
	URL         string
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
func NewCmdGetApps(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetAppsOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
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
	return cmd
}

// Run implements this command
func (o *GetAppsOptions) Run() error {
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}

	apps, err := o.getAppData(kubeClient)
	if err != nil {
		return err
	}

	if len(apps.Items) == 0 {
		log.Infof("No Apps found\n")
		return nil
	}

	table := o.generateTable(apps, kubeClient)

	table.Render()
	return nil
}

func (o *GetAppsOptions) generateTable(apps *v1.AppList, kubeClient kubernetes.Interface) table.Table {
	table := o.generateTableHeaders(apps)
	for _, app := range apps.Items {
		row := []string{app.Annotations[helm.AnnotationChartName], app.Labels[helm.LabelReleaseChartVersion], app.Annotations[helm.AnnotationAppDescription], app.Annotations[helm.AnnotationAppRepository]}
		table.AddRow(row...)
	}
	return table
}

func (o *GetAppsOptions) getAppData(kubeClient kubernetes.Interface) (apps *v1.AppList, err error) {
	client, currentNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "getting jx client")
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return nil, errors.Wrap(err, "getting current dev namespace")
	}
	listOptions := metav1.ListOptions{}
	if len(o.Args) > 0 {
		selector := fmt.Sprintf(helm.LabelAppName+" in (%s)", strings.Join(o.Args[:], ", "))
		listOptions.LabelSelector = selector
	}
	apps, err = client.JenkinsV1().Apps(ns).List(listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "listing apps")
	}
	return apps, nil
}

func (o *GetAppsOptions) generateTableHeaders(apps *v1.AppList) table.Table {
	t := o.createTable()
	titles := []string{"Name", "Version", "Description", "Chart Repository"}
	t.AddRow(titles...)
	return t
}
