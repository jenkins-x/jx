package cmd

import (
	"io"
	"strings"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"fmt"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/addon"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	upgradeAppsLong = templates.LongDesc(`
		Upgrades Apps to newer releases
`)

	upgradeAppsExample = templates.Examples(`
		# Upgrade all apps 
		jx upgrade apps
 
        # Upgrade a specific app
        jx upgrade app cheese
	`)
)

// UpgradeAppsOptions the options for the create spring command
type UpgradeAppsOptions struct {
	AddOptions

	GitOps bool
	DevEnv *jenkinsv1.Environment

	Repo     string
	Alias    string
	Username string
	Password string

	Version string
	All     bool

	Namespace string
	Set       []string

	// for testing
	FakePullRequests CreateEnvPullRequestFn

	// allow git to be configured externally before a PR is created
	ConfigureGitCallback ConfigureGitFolderFn

	InstallFlags InstallFlags
}

// NewCmdUpgradeApps defines the command
func NewCmdUpgradeApps(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	o := &UpgradeAppsOptions{
		AddOptions: AddOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "apps",
		Short:   "Upgrades any Apps to the latest release",
		Aliases: []string{"app"},
		Long:    upgradeAppsLong,
		Example: upgradeAppsExample,
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&o.BatchMode, optionBatchMode, "b", false, "In batch mode the command never prompts for user input")
	cmd.Flags().BoolVarP(&o.Verbose, optionVerbose, "", false, "Enable verbose logging")
	cmd.Flags().StringVarP(&o.Version, "username", "", "",
		"The username for the repository")
	cmd.Flags().StringVarP(&o.Version, "password", "", "",
		"The password for the repository")
	cmd.Flags().StringVarP(&o.Repo, "repository", "", "",
		"The repository from which the app should be installed")
	cmd.Flags().StringVarP(&o.Alias, "alias", "", "", "An alias to use for the app [--gitops]")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "",
		"The chart version to install [--gitops]")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "The Namespace to promote to [--no-gitops]")
	cmd.Flags().StringArrayVarP(&o.Set, "set", "s", []string{},
		"The Helm parameters to pass in while upgrading [--no-gitops]")

	return cmd
}

// Run implements the command
func (o *UpgradeAppsOptions) Run() error {
	o.GitOps, o.DevEnv = o.GetDevEnv()
	if o.Repo == "" {
		o.Repo = o.DevEnv.Spec.TeamSettings.AppsRepository
	}

	if o.GitOps {
		msg := "Unable to specify --%s when using GitOps for your dev environment"
		if o.Namespace != "" {
			return util.InvalidOptionf(optionNamespace, o.ReleaseName, msg, optionNamespace)
		}
		if len(o.Set) > 0 {
			return util.InvalidOptionf(optionSet, o.ReleaseName, msg, optionSet)
		}
	}
	if !o.GitOps {
		msg := "Unable to specify --%s when NOT using GitOps for your dev environment"
		if o.Alias != "" {
			return util.InvalidOptionf(optionAlias, o.ReleaseName, msg, optionAlias)
		}
		if o.Version != "" {
			return util.InvalidOptionf(optionVersion, o.ReleaseName, msg, optionVersion)
		}
	}

	if o.GitOps {
		err := o.createPRs()
		if err != nil {
			return err
		}
	} else {
		err := o.upgradeApps()
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *UpgradeAppsOptions) createPRs() error {
	var branchNameText string
	var title string
	var message string

	if len(o.Args) > 1 {
		return fmt.Errorf("specify at most one app to upgrade")
	} else if len(o.Args) == 0 {
		o.All = true
		branchNameText = fmt.Sprintf("upgrade-all-apps")
		title = fmt.Sprintf("Upgrade all apps")
		message = fmt.Sprintf("Upgrade all apps:\n")
	}
	upgraded := false
	modifyChartFn := func(requirements *helm.Requirements, metadata *chart.Metadata, values map[string]interface{},
		templates map[string]map[string]interface{}) error {
		for _, d := range requirements.Dependencies {
			upgrade := false
			// We need to ignore the platform
			if d.Name == "jenkins-x-platform" {
				upgrade = false
			} else if o.All {
				upgrade = true
			} else {
				if d.Name == o.Args[0] && d.Alias == o.Alias {
					upgrade = true

				}
			}
			if upgrade {
				upgraded = true
				version := o.Version
				if o.All || version == "" {
					var err error
					version, err = helm.GetLatestVersion(d.Name, o.Repo, o.Username, o.Password, o.Helm())
					if err != nil {
						return err
					}
					if o.Verbose {
						log.Infof("No version specified so using latest version which is %s\n", util.ColorInfo(version))
					}
				}
				// Do the upgrade
				oldVersion := d.Version
				d.Version = version
				if !o.All {
					branchNameText = fmt.Sprintf("upgrade-app-%s-%s", o.Args[0], version)
					title = fmt.Sprintf("Upgrade %s to %s", o.Args[0], version)
					message = fmt.Sprintf("Upgrade %s from %s to %s", o.Args[0], oldVersion, version)
				} else {
					message = fmt.Sprintf("%s\n* %s from %s to %s", message, d.Name, oldVersion, version)
				}
			}
		}
		return nil
	}

	if o.FakePullRequests != nil {
		var err error
		_, err = o.FakePullRequests(o.DevEnv, modifyChartFn, branchNameText, title, message,
			nil)
		if err != nil {
			return err
		}
	} else {
		var err error
		_, err = o.createEnvironmentPullRequest(o.DevEnv, modifyChartFn, &branchNameText, &title,
			&message,
			nil, o.ConfigureGitCallback)
		if err != nil {
			return err
		}
	}

	if !upgraded {
		log.Infof("No upgrades available\n")
	}
	return nil
}

func (o *UpgradeAppsOptions) upgradeApps() error {
	err := o.Helm().UpdateRepo()
	if err != nil {
		return err
	}
	ns := o.Namespace
	if ns == "" {
		_, ns, err = o.JXClientAndDevNamespace()
		if err != nil {
			return err
		}
	}

	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	o.devNamespace, _, err = kube.GetDevNamespace(client, ns)
	if err != nil {
		return err
	}

	appConfig, err := addon.LoadAddonsConfig()
	if err != nil {
		return err
	}
	appEnabled := map[string]bool{}
	for _, app := range appConfig.Addons {
		if app.Enabled {
			appEnabled[app.Name] = true
		}
	}
	statusMap, err := o.Helm().StatusReleases(ns)
	if err != nil {
		log.Warnf("Failed to find Helm installs: %s\n", err)
	}

	charts := kube.AddonCharts
	keys := []string{}
	if len(o.Args) > 0 {
		for _, k := range o.Args {
			chart := charts[k]
			if chart == "" {
				return errors.Wrapf(err, "failed to match app %s", k)
			}
			keys = append(keys, k)
		}
	} else {
		keys = util.SortedMapKeys(charts)
	}

	for _, k := range keys {
		app := charts[k]
		status := statusMap[k].Status
		name := k
		if name == k {
			name = "kube-cd"
		}
		if status != "" {
			log.Infof("Upgrading %s app %s...\n", util.ColorInfo(name), util.ColorInfo(app))

			valueFiles := []string{}
			valueFiles, err = helm.AppendMyValues(valueFiles)
			if err != nil {
				return errors.Wrap(err, "failed to append the myvalues.yaml file")
			}

			values := []string{}
			for _, vs := range o.Set {
				values = append(values, strings.Split(vs, ",")...)
			}

			config := &v1.ConfigMap{}
			plugins := &v1.ConfigMap{}
			if k == kube.DefaultProwReleaseName {
				// lets backup any Prow config as we should never replace this, eventually we'll move config to a git repo so this is temporary
				config, plugins, err = o.backupConfigs()
				if err != nil {
					return errors.Wrap(err, "backing up the configuration")
				}
			}
			err = o.Helm().UpgradeChart(app, k, ns, nil, false, nil, false, false, values, valueFiles, "",
				o.Username, o.Password)
			if err != nil {
				log.Warnf("Failed to upgrade %s app %s: %v\n", name, app, err)
			}

			if k == kube.DefaultProwReleaseName {
				err = o.restoreConfigs(config, plugins)
				if err != nil {
					return err
				}
			}

			log.Infof("Upgraded %s app %s\n", util.ColorInfo(name), util.ColorInfo(app))
		}
	}
	return nil
}

func (o *UpgradeAppsOptions) restoreConfigs(config *v1.ConfigMap, plugins *v1.ConfigMap) error {
	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	var err1 error
	if config != nil {
		_, err = client.CoreV1().ConfigMaps(o.devNamespace).Get("config", metav1.GetOptions{})
		if err != nil {
			_, err = client.CoreV1().ConfigMaps(o.devNamespace).Create(config)
			if err != nil {
				b, _ := yaml.Marshal(config)
				err1 = fmt.Errorf("error restoring config %s\n", string(b))
			}
		}
	}
	if plugins != nil {
		_, err = client.CoreV1().ConfigMaps(o.devNamespace).Get("plugins", metav1.GetOptions{})
		if err != nil {
			_, err = client.CoreV1().ConfigMaps(o.devNamespace).Create(plugins)
			if err != nil {
				b, _ := yaml.Marshal(plugins)
				err = fmt.Errorf("%v/nerror restoring plugins %s\n", err1, string(b))
			}
		}
	}
	return err
}

func (o *UpgradeAppsOptions) backupConfigs() (*v1.ConfigMap, *v1.ConfigMap, error) {
	client, err := o.KubeClient()
	if err != nil {
		return nil, nil, err
	}
	config, _ := client.CoreV1().ConfigMaps(o.devNamespace).Get("config", metav1.GetOptions{})
	plugins, _ := client.CoreV1().ConfigMaps(o.devNamespace).Get("plugins", metav1.GetOptions{})
	config = config.DeepCopy()
	config.ResourceVersion = ""
	plugins = plugins.DeepCopy()
	plugins.ResourceVersion = ""
	return config, plugins, nil
}
