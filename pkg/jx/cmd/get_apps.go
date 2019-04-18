package cmd

import (
	"encoding/json"
	"fmt"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// GetAppsOptions containers the CLI options
type GetAppsOptions struct {
	GetOptions
	Namespace  string
	ShowStatus bool
	GitOps     bool
	DevEnv     *v1.Environment
}

type appsResult struct {
	AppOutput []appOutput `json:"items"`
}

type appOutput struct {
	Name            string `yaml:"appName" json:"appName"`
	Version         string `yaml:"version" json:"version"`
	Description     string `yaml:"description" json:"description"`
	ChartRepository string `yaml:"chartRepository" json:"chartRepository"`
	Status          string `yaml:"status" json:"status"`
}

// HelmOutput is the representation of the Helm status command
type HelmOutput struct {
	helmInfo `yaml:"info" json:"info"`
}

type helmInfo struct {
	helmInfoStatus `yaml:"status" json:"status"`
}

type helmInfoStatus struct {
	Resources string `json:"resources" json:"resources"`
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
		
		# Display detailed status info about the app called cheese
		jx get app cheese --status

		# Display detailed status info about the app called cheese in 'json' format
		jx get app cheese --status -o json

		# Display details about the app called cheese in 'yaml' format
		jx get app cheese -o yaml
	`)
)

// NewCmdGetApps creates the new command for: jx get version
func NewCmdGetApps(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetAppsOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "apps",
		Short:   "Display one or more installed Apps",
		Aliases: []string{"app"},
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
	cmd.Flags().StringVarP(&options.Namespace, optionNamespace, "n", "", "The namespace where you want to search the apps in")
	cmd.Flags().BoolVarP(&options.ShowStatus, "status", "s", false, "Shows detailed information about the state of the app")
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
		if len(o.Args) > 0 {
			return errors.New("No Apps found")
		}

		fmt.Fprint(o.Out, "No Apps found\n")
		return nil
	}

	if o.ShowStatus {
		differentAppsInSlice, err := checkDifferentAppsInAppsSlice(apps)
		if err != nil {
			fmt.Fprintln(o.Out, "There was a problem trying to show the status of the app: ", err)
			return nil
		}
		if differentAppsInSlice {
			fmt.Fprint(o.Out, "Different apps provided. Provide only one app to check the status of\n")
			return nil
		}
		return o.generateAppStatusOutput(&apps.Items[0])
	}

	if o.Output != "" {
		appsResult := o.generateTableFormatted(apps)
		return o.renderResult(appsResult, o.Output)
	}
	table := o.generateTable(apps, kubeClient)
	table.Render()
	return nil
}

func (o *GetAppsOptions) generateAppStatusOutput(app *v1.App) error {
	output, err := o.Helm().StatusReleaseWithOutput(o.Namespace, app.Labels[helm.LabelAppName], "json")
	if err != nil {
		return err
	}
	return o.printHelmResourcesWithFormat(output)
}

func (o *GetAppsOptions) generateTableFormatted(apps *v1.AppList) appsResult {
	releasesMap, err := o.Helm().StatusReleases(o.Namespace)
	if err != nil {
		log.Warnf("There was a problem obtaining the app status: %v\n", err)
	}
	results := appsResult{}
	for _, app := range apps.Items {
		if app.Labels != nil {
			name := app.Labels[helm.LabelAppName]
			if name != "" && app.Annotations != nil {
				var status string
				if releasesMap != nil {
					status = releasesMap[name].Status
				}
				results.AppOutput = append(results.AppOutput, appOutput{
					Name:            name,
					Version:         app.Labels[helm.LabelAppVersion],
					Description:     app.Annotations[helm.AnnotationAppDescription],
					ChartRepository: app.Annotations[helm.AnnotationAppRepository],
					Status:          status,
				})
			}
		}
	}
	return results
}

func (o *GetAppsOptions) generateTable(apps *v1.AppList, kubeClient kubernetes.Interface) table.Table {
	table := o.generateTableHeaders(apps)
	releasesMap, err := o.Helm().StatusReleases(o.Namespace)
	if err != nil {
		log.Warnf("There was a problem obtaining the app status: %v\n", err)
	}
	for _, app := range apps.Items {
		if app.Labels != nil {
			name := app.Labels[helm.LabelAppName]
			if name != "" && app.Annotations != nil {
				version := app.Labels[helm.LabelAppVersion]
				description := app.Annotations[helm.AnnotationAppDescription]
				repository := app.Annotations[helm.AnnotationAppRepository]
				var status string
				if releasesMap != nil {
					status = releasesMap[name].Status
				}
				namespace := app.Namespace
				row := []string{name, version, repository, namespace, status, description}
				table.AddRow(row...)
			}
		}

	}
	return table
}

func (o *GetAppsOptions) generateTableHeaders(apps *v1.AppList) table.Table {
	t := o.CreateTable()
	t.Out = o.CommonOptions.Out
	titles := []string{"Name", "Version", "Chart Repository", "Namespace", "Status", "Description"}
	t.AddRow(titles...)
	return t
}

func (o *GetAppsOptions) printHelmResourcesWithFormat(helmOutputJSON string) error {
	h := HelmOutput{}
	err := json.Unmarshal([]byte(helmOutputJSON), &h)
	if err != nil {
		return err
	}
	if o.Output == "" {
		fmt.Fprintln(o.Out, h.helmInfoStatus.Resources)
		return nil
	}
	return o.renderResult(h, o.Output)

}

func checkDifferentAppsInAppsSlice(apps *v1.AppList) (bool, error) {
	if len(apps.Items) == 0 {
		return false, errors.New("no apps were provided to check the status of")
	}
	x, a := apps.Items[0], apps.Items[1:]
	for _, app := range a {
		if app.Name != x.Name {
			return true, nil
		}
	}
	return false, nil
}
