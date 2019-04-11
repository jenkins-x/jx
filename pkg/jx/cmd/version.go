package cmd

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/util/system"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	jxChartPrefix = "jenkins-x-platform-"
)

type VersionOptions struct {
	*opts.CommonOptions

	Container      string
	Namespace      string
	HelmTLS        bool
	NoVersionCheck bool
}

func NewCmdVersion(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &VersionOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	/*
		cmd.Flags().BoolP("client", "c", false, "Client version only (no server required).")
		cmd.Flags().BoolP("short", "", false, "Print just the version number.")
	*/
	cmd.Flags().MarkShorthandDeprecated("client", "please use --client instead.")
	cmd.Flags().BoolVarP(&options.HelmTLS, "helm-tls", "", false, "Whether to use TLS with helm")
	cmd.Flags().BoolVarP(&options.NoVersionCheck, "no-version-check", "n", false, "Disable checking of version upgrade checks")
	return cmd
}

func (o *VersionOptions) Run() error {
	info := util.ColorInfo
	table := o.CreateTable()
	table.AddRow("NAME", "VERSION")
	table.AddRow("jx", info(version.GetVersion()))

	// Jenkins X version
	output, err := o.Helm().ListCharts()
	if err != nil {
		logrus.Warnf("Failed to find helm installs: %s\n", err)
	} else {
		for _, line := range strings.Split(output, "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) > 4 && strings.TrimSpace(fields[0]) == "jenkins-x" {
				for _, f := range fields[4:] {
					f = strings.TrimSpace(f)
					if strings.HasPrefix(f, jxChartPrefix) {
						chart := strings.TrimPrefix(f, jxChartPrefix)
						table.AddRow("jenkins x platform", info(chart))
					}
				}
			}
		}
	}

	// Kubernetes version
	client, err := o.KubeClient()
	if err != nil {
		logrus.Warnf("Failed to connect to Kubernetes: %s\n", err)
	} else {
		serverVersion, err := client.Discovery().ServerVersion()
		if err != nil {
			logrus.Warnf("Failed to get Kubernetes server version: %s\n", err)
		} else if serverVersion != nil {
			table.AddRow("Kubernetes cluster", info(serverVersion.String()))
		}
	}

	// kubectl version
	output, err = o.GetCommandOutput("", "kubectl", "version", "--short")
	if err != nil {
		logrus.Warnf("Failed to get kubectl version: %s\n", err)
	} else {
		for i, line := range strings.Split(output, "\n") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				v := fields[2]
				if v != "" {
					switch i {
					case 0:
						table.AddRow("kubectl", info(v))
					case 1:
						// Ignore K8S server details as we have these above
					}
				}
			}
		}
	}

	// helm version
	output, err = o.Helm().Version(o.HelmTLS)
	if err != nil {
		logrus.Warnf("Failed to get helm version: %s\n", err)
	} else {
		helmBinary, noTiller, helmTemplate, _ := o.TeamHelmBin()
		if helmBinary == "helm3" || noTiller || helmTemplate {
			table.AddRow("helm client", info(output))
		} else {
			for i, line := range strings.Split(output, "\n") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					v := fields[1]
					if v != "" {
						switch i {
						case 0:
							table.AddRow("helm client", info(v))
						case 1:
							table.AddRow("helm server", info(v))
						}
					}
				}
			}
		}
	}

	// git version
	version, err := o.Git().Version()
	if err != nil {
		logrus.Warnf("Failed to get git version: %s\n", err)
	} else {
		table.AddRow("git", info(version))
	}

	// os version
	version, err = o.GetOsVersion()
	if err != nil {
		logrus.Warnf("Failed to get OS version: %s\n", err)
	} else {
		table.AddRow("Operating System", info(version))
	}

	table.Render()

	if !o.NoVersionCheck {
		newVersion, err := o.GetLatestJXVersion()
		if err != nil {
			return errors.Wrap(err, "getting latest jx version")
		}
		update, err := o.ShouldUpdate(newVersion)
		if err != nil {
			return errors.Wrap(err, "checking version")
		}
		if update {
			return o.upgradeCli(newVersion)
		}
	}
	return nil
}

// ShouldUpdate checks if CLI version should be updated
func (o *VersionOptions) ShouldUpdate(newVersion semver.Version) (bool, error) {
	currentVersion, err := version.GetSemverVersion()
	if err != nil {
		return false, err
	}

	if newVersion.GT(currentVersion) {
		// Do not ask to update if we are using a dev build...
		for _, x := range currentVersion.Pre {
			if x.VersionStr == "dev" {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

func (o *VersionOptions) upgradeCli(newVersion semver.Version) error {
	app := util.ColorInfo("jx")
	logrus.Warnf("\nA new %s version is available: %s\n", app, util.ColorInfo(newVersion.String()))
	if o.BatchMode {
		logrus.Warnf("To upgrade to this new version use: %s\n", util.ColorInfo("jx upgrade cli"))
	} else {
		message := fmt.Sprintf("Would you like to upgrade to the new %s version?", app)
		if util.Confirm(message, true, "Please indicate if you would like to upgrade the binary version.", o.In, o.Out, o.Err) {
			options := &UpgradeCLIOptions{
				CreateOptions: CreateOptions{
					CommonOptions: o.CommonOptions,
				},
			}
			return options.Run()
		}
	}
	return nil
}

// GetOsVersion returns a human friendly string of the current OS
// in the case of an error this still returns a valid string for the details that can be found.
func (o *VersionOptions) GetOsVersion() (string, error) {
	return system.GetOsVersion()
}
