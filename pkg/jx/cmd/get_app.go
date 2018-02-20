package cmd

import (
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetVersionOptions containers the CLI options
type GetVersionOptions struct {
	CommonOptions
}

var (
	get_version_long = templates.LongDesc(`
		Display applications across environments.
`)

	get_version_example = templates.Examples(`
		# List applications for all environments
		jx get app
	`)
)

// NewCmdGetVersion creates the new command for: jx get version
func NewCmdGetVersion(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetVersionOptions{
		CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "apps",
		Short:   "Display one or many Applications and their versions",
		Aliases: []string{"app", "version", "versions"},
		Long:    get_version_long,
		Example: get_version_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	return cmd
}

type EnvApps struct {
	Environment v1.Environment
	Apps        map[string]v1beta1.Deployment
}

// Run implements this command
func (o *GetVersionOptions) Run() error {
	f := o.Factory
	client, currentNs, err := f.CreateJXClient()
	if err != nil {
		return err
	}
	kubeClient, _, err := o.Factory.CreateClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return err
	}
	envList, err := client.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	kube.SortEnvironments(envList.Items)

	namespaces := []string{}
	envApps := []EnvApps{}
	envNames := []string{}
	apps := []string{}
	for _, env := range envList.Items {
		ens := env.Spec.Namespace
		namespaces = append(namespaces, ens)
		if ens != "" && env.Name != kube.LabelValueDevEnvironment {
			envNames = append(envNames, env.Name)
			m, err := kube.GetDeployments(kubeClient, ens)
			if err == nil {
				envApp := EnvApps{
					Environment: env,
					Apps:        map[string]v1beta1.Deployment{},
				}
				envApps = append(envApps, envApp)
				for k, d := range m {
					appName := kube.GetAppName(k, ens)
					envApp.Apps[appName] = d
					if util.StringArrayIndex(apps, appName) < 0 {
						apps = append(apps, appName)
					}
				}
			}
		}
	}
	util.ReverseStrings(namespaces)
	if len(apps) == 0 {
		o.Printf("No applications found in environments %s\n", strings.Join(envNames, ", "))
		return nil
	}
	sort.Strings(apps)

	table := o.CreateTable()
	titles := []string{"APPLICATION"}
	for _, ea := range envApps {
		titles = append(titles, strings.ToUpper(ea.Environment.Name), "PODS")
	}
	table.AddRow(titles...)

	for _, appName := range apps {
		row := []string{appName}
		for _, ea := range envApps {
			version := ""
			pods := ""
			d := ea.Apps[appName]
			version = kube.GetVersion(&d.ObjectMeta)
			replicas := ""
			ready := d.Status.ReadyReplicas
			if d.Spec.Replicas != nil && ready > 0 {
				replicas = formatInt32(*d.Spec.Replicas)
				pods = formatInt32(ready) + "/" + replicas
			}
			row = append(row, version, pods)
		}
		table.AddRow(row...)
	}
	table.Render()
	return nil
}
