package cmd

import (
	"io"

	"fmt"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/addon"
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
	upgradeAddonsLong = templates.LongDesc(`
		Upgrades the Jenkins X platform if there is a newer release
`)

	upgradeAddonsExample = templates.Examples(`
		# Upgrades the Jenkins X platform 
		jx upgrade addons
	`)
)

// UpgradeAddonsOptions the options for the create spring command
type UpgradeAddonsOptions struct {
	CreateOptions

	Namespace string
	Set       string

	InstallFlags InstallFlags
}

// NewCmdUpgradeAddons defines the command
func NewCmdUpgradeAddons(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpgradeAddonsOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "addons",
		Short:   "Upgrades any Addons added to Jenkins X if there are any new releases available",
		Aliases: []string{"addon"},
		Long:    upgradeAddonsLong,
		Example: upgradeAddonsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The Namespace to promote to")
	cmd.Flags().StringVarP(&options.Set, "set", "s", "", "The Helm parameters to pass in while upgrading")

	options.addCommonFlags(cmd)
	options.InstallFlags.addCloudEnvOptions(cmd)

	cmd.AddCommand(NewCmdUpgradeAddonProw(f, in, out, errOut))

	return cmd
}

func (options *UpgradeAddonsOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "", "", "The Namespace to upgrade")
	cmd.Flags().StringVarP(&options.Set, "set", "s", "", "The Helm parameters to pass in while upgrading")

}

// Run implements the command
func (o *UpgradeAddonsOptions) Run() error {
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

	addonConfig, err := addon.LoadAddonsConfig()
	if err != nil {
		return err
	}
	addonEnabled := map[string]bool{}
	for _, addon := range addonConfig.Addons {
		if addon.Enabled {
			addonEnabled[addon.Name] = true
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
				return errors.Wrapf(err, "failed to match addon %s", k)
			}
			keys = append(keys, k)
		}
	} else {
		keys = util.SortedMapKeys(charts)
	}

	for _, k := range keys {
		chart := charts[k]
		status := statusMap[k].Status
		name := k
		if name == k {
			name = "kube-cd"
		}
		if status != "" {
			log.Infof("Upgrading %s chart %s...\n", util.ColorInfo(name), util.ColorInfo(chart))

			valueFiles := []string{}
			valueFiles, err = helm.AppendMyValues(valueFiles)
			if err != nil {
				return errors.Wrap(err, "failed to append the myvalues.yaml file")
			}

			values := []string{}
			if o.Set != "" {
				values = append(values, o.Set)
			}

			config := &v1.ConfigMap{}
			plugins := &v1.ConfigMap{}
			if k == kube.DefaultProwReleaseName {
				// lets backup any Prow config as we should never replace this, eventually we'll move config to a git repo so this is temporary
				config, plugins, err = o.backupConfigs()
				if err != nil {
					return errors.Wrap(err, "backing up the prow config")
				}
			}
			err = o.Helm().UpgradeChart(chart, k, ns, nil, false, nil, false, false, values, valueFiles, "", "", "")
			if err != nil {
				log.Warnf("Failed to upgrade %s chart %s: %v\n", name, chart, err)
			}

			if k == kube.DefaultProwReleaseName {
				err = o.restoreConfigs(config, plugins)
				if err != nil {
					return err
				}
			}

			log.Infof("Upgraded %s chart %s\n", util.ColorInfo(name), util.ColorInfo(chart))
		}
	}
	return nil
}

func (o *UpgradeAddonsOptions) restoreConfigs(config *v1.ConfigMap, plugins *v1.ConfigMap) error {
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

func (o *UpgradeAddonsOptions) backupConfigs() (*v1.ConfigMap, *v1.ConfigMap, error) {
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
