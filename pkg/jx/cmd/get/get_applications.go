package get

import (
	"os/user"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/jenkins-x/jx/pkg/kserving"
	"github.com/jenkins-x/jx/pkg/kube/services"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/flagger"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	kserve "github.com/knative/serving/pkg/client/clientset/versioned"
	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetApplicationsOptions containers the CLI options
type GetApplicationsOptions struct {
	*opts.CommonOptions

	Namespace   string
	Environment string
	HideUrl     bool
	HidePod     bool
	Previews    bool

	Results GetApplicationsResults
}

// GetApplicationsResults contains the data result from invoking this command
type GetApplicationsResults struct {
	EnvApps  []EnvApps
	EnvNames []string

	// Applications is a map indexed by the application name then the environment name
	Applications map[string]map[string]*ApplicationEnvironmentInfo
}

// EnvApps contains data about app deployments in an environment
type EnvApps struct {
	Environment v1.Environment
	Apps        map[string]v1beta1.Deployment
}

// ApplicationEnvironmentInfo contains the results of an app for an environment
type ApplicationEnvironmentInfo struct {
	Deployment  *v1beta1.Deployment
	Environment *v1.Environment
	Version     string
	URL         string
}

var (
	get_version_long = templates.LongDesc(`
		Display applications across environments.
`)

	get_version_example = templates.Examples(`
		# List applications, their URL and pod counts for all environments
		jx get applications

		# List applications only in the Staging environment
		jx get applications -e staging

		# List applications only in the Production environment
		jx get applications -e production

		# List applications only in a specific namespace
		jx get applications -n jx-staging

		# List applications hiding the URLs
		jx get applications -u

		# List applications just showing the versions (hiding urls and pod counts)
		jx get applications -u -p
	`)
)

// NewCmdGetApplications creates the new command for: jx get version
func NewCmdGetApplications(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetApplicationsOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "applications",
		Short:   "Display one or more Applications and their versions",
		Aliases: []string{"application", "version", "versions"},
		Long:    get_version_long,
		Example: get_version_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.HideUrl, "url", "u", false, "Hide the URLs")
	cmd.Flags().BoolVarP(&options.HidePod, "pod", "p", false, "Hide the pod counts")
	cmd.Flags().BoolVarP(&options.Previews, "preview", "w", false, "Show preview environments only")
	cmd.Flags().StringVarP(&options.Environment, "env", "e", "", "Filter applications in the given environment")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Filter applications in the given namespace")
	return cmd
}

// Run implements this command
func (o *GetApplicationsOptions) Run() error {
	kubeClient, err := o.KubeClient()
	if err != nil {
		return err
	}
	kserveClient, _, err := o.KnativeServeClient()
	if err != nil {
		return err
	}

	_, envApps, envNames, apps, err := o.getAppData(kubeClient)
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		log.Logger().Infof("No applications found in environments %s", strings.Join(envNames, ", "))
		return nil
	}
	sort.Strings(apps)

	o.Results.EnvApps = envApps
	o.Results.EnvNames = envNames

	table := o.generateTable(apps, envApps, kubeClient, kserveClient)

	table.Render()
	return nil
}

func (o *GetApplicationsOptions) generateTable(apps []string, envApps []EnvApps, kubeClient kubernetes.Interface, kserveClient kserve.Interface) table.Table {
	table := o.generateTableHeaders(envApps)

	appEnvMap := map[string]map[string]*ApplicationEnvironmentInfo{}
	for _, appName := range apps {
		row := []string{appName}
		for _, ea := range envApps {
			version := ""
			d, ok := ea.Apps[appName]
			if len(ea.Apps) > 0 {
				if ok {
					appMap := appEnvMap[appName]
					if appMap == nil {
						appMap = map[string]*ApplicationEnvironmentInfo{}
						appEnvMap[appName] = appMap
					}
					version = kube.GetVersion(&d.ObjectMeta)
					url := ""
					if !o.HideUrl {
						names := []string{appName}
						if d.Name != appName {
							names = append(names, d.Name)
						}
						for _, name := range names {
							url, _ = services.FindServiceURL(kubeClient, d.Namespace, name)
							if url != "" {
								break
							}
							url2, svc, _ := kserving.FindServiceURL(kserveClient, kubeClient, d.Namespace, name)
							if url2 != "" {
								url = url2
								if svc != nil {
									svcVersion := kube.GetVersion(&svc.ObjectMeta)
									if svcVersion != "" {
										version = svcVersion
									}
								}
								break
							}
						}
						if url == "" {
							// handle helm3
							chart, ok := d.Labels["chart"]
							if ok {
								idx := strings.LastIndex(chart, "-")
								if idx > 0 {
									svcName := chart[0:idx]
									if svcName != appName && svcName != d.Name {
										url, _ = services.FindServiceURL(kubeClient, d.Namespace, svcName)
									}
								}
							}
						}
					}

					appEnvInfo := &ApplicationEnvironmentInfo{
						Deployment:  &d,
						Environment: ea.Environment.DeepCopy(),
						Version:     version,
						URL:         url,
					}
					appMap[ea.Environment.Name] = appEnvInfo

					if ea.Environment.Spec.Kind != v1.EnvironmentKindTypePreview {
						row = append(row, version)
					}
					if !o.HidePod {
						pods := ""
						replicas := ""
						ready := d.Status.ReadyReplicas
						if d.Spec.Replicas != nil && ready > 0 {
							replicas = formatInt32(*d.Spec.Replicas)
							pods = formatInt32(ready) + "/" + replicas
						}
						row = append(row, pods)
					}
					if !o.HideUrl {
						row = append(row, url)
					}
				} else {
					if ea.Environment.Spec.Kind != v1.EnvironmentKindTypePreview {
						row = append(row, "")
					}
					if !o.HidePod {
						row = append(row, "")
					}
					if !o.HideUrl {
						row = append(row, "")
					}
				}
			}
		}
		table.AddRow(row...)
	}
	o.Results.Applications = appEnvMap
	return table
}

func (o *GetApplicationsOptions) getAppData(kubeClient kubernetes.Interface) (namespaces []string, envApps []EnvApps, envNames, apps []string, err error) {
	client, currentNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "getting jx client")
	}
	u, err := user.Current()
	if err != nil {
		log.Logger().Warnf("could not find the current user name %s", err.Error())
	}
	username := "uknown"
	if u != nil {
		username = u.Username
	}

	ns, _, err := kube.GetDevNamespace(kubeClient, currentNs)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "getting current dev namespace")
	}
	envList, err := client.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "listing environments")
	}
	kube.SortEnvironments(envList.Items)

	namespaces, envNames, apps = []string{}, []string{}, []string{}
	envApps = []EnvApps{}
	for _, env := range envList.Items {
		isPreview := env.Spec.Kind == v1.EnvironmentKindTypePreview
		shouldShow := isPreview
		if !o.Previews {
			shouldShow = !shouldShow
		}
		if shouldShow &&
			(o.Environment == "" || o.Environment == env.Name) &&
			(o.Namespace == "" || o.Namespace == env.Spec.Namespace) {
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
						// lets use the logical service name from kserve
						if d.Labels != nil {
							serviceName := d.Labels[kserving.ServiceLabel]
							if serviceName != "" {
								k = serviceName
							}
						}
						appName := kube.GetAppName(k, ens)
						if env.Spec.Kind == v1.EnvironmentKindTypeEdit {
							if appName == kube.DeploymentExposecontrollerService || env.Spec.PreviewGitSpec.User.Username != username {
								continue
							}
							appName = kube.GetEditAppName(appName)
						} else if env.Spec.Kind == v1.EnvironmentKindTypePreview {
							appName = env.Spec.PullRequestURL
						}

						// Ignore flagger canary auxiliary deployments
						if flagger.IsCanaryAuxiliaryDeployment(d) {
							continue
						}

						envApp.Apps[appName] = d
						if util.StringArrayIndex(apps, appName) < 0 {
							apps = append(apps, appName)
						}
					}
				}
			}
		}
	}
	return
}

func (o *GetApplicationsOptions) generateTableHeaders(envApps []EnvApps) table.Table {
	t := o.CreateTable()
	title := "APPLICATION"
	if o.Previews {
		title = "PULL REQUESTS"
	}
	titles := []string{title}
	for _, ea := range envApps {
		envName := ea.Environment.Name
		if len(ea.Apps) > 0 {
			if ea.Environment.Spec.Kind == v1.EnvironmentKindTypeEdit {
				envName = "Edit"
			}
			if ea.Environment.Spec.Kind != v1.EnvironmentKindTypePreview {
				titles = append(titles, strings.ToUpper(envName))
			}
			if !o.HidePod {
				titles = append(titles, "PODS")
			}
			if !o.HideUrl {
				titles = append(titles, "URL")
			}
		}
	}
	t.AddRow(titles...)
	return t
}
