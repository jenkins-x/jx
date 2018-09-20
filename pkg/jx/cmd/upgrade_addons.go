package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/addon"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	upgradeAddonsLong = templates.LongDesc(`
		Upgrades the Jenkins X platform if there is a newer release
`)

	upgradeAddonsExample = templates.Examples(`
		# Upgrades the Jenkins X platform 
		jx upgrade platform
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
	cmd.Flags().StringVarP(&options.Set, "set", "s", "", "The helm parameters to pass in while upgrading")

	options.addCommonFlags(cmd)
	options.InstallFlags.addCloudEnvOptions(cmd)

	return cmd
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
	statusMap, err := o.Helm().StatusReleases()
	if err != nil {
		log.Warnf("Failed to find helm installs: %s\n", err)
	}

	charts := kube.AddonCharts

	keys := util.SortedMapKeys(charts)
	for _, k := range keys {
		chart := charts[k]
		status := statusMap[k]
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

			err = o.Helm().UpgradeChart(chart, k, ns, nil, false, nil, false, false, values, valueFiles)
			if err != nil {
				return errors.Wrapf(err, "Failed to upgrade %s chart %s\n", name, chart)
			}
			log.Infof("Upgraded %s chart %s\n", util.ColorInfo(name), util.ColorInfo(chart))
		}
	}
	return nil
}
