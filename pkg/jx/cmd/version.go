package cmd

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/spf13/cobra"
)

const (
	jxChartPrefix = "jenkins-x-platform-"
)

type VersionOptions struct {
	CommonOptions

	Container      string
	Namespace      string
	HelmTLS        bool
	NoVersionCheck bool
}

func NewCmdVersion(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &VersionOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	/*
		cmd.Flags().BoolP("client", "c", false, "Client version only (no server required).")
		cmd.Flags().BoolP("short", "", false, "Print just the version number.")
	*/
	options.addCommonFlags(cmd)

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

	helmBin, err := o.TeamHelmBin()
	if err != nil {
		return err
	}

	// Jenkins X version
	output, err := o.getCommandOutput("", helmBin, "list")
	if err != nil {
		o.warnf("Failed to find helm installs: %s\n", err)
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

	// kubernetes version
	client, _, err := o.KubeClient()
	if err != nil {
		o.warnf("Failed to connect to kubernetes: %s\n", err)
	} else {
		serverVersion, err := client.Discovery().ServerVersion()
		if err != nil {
			o.warnf("Failed to get kubernetes server version: %s\n", err)
		} else if serverVersion != nil {
			table.AddRow("kubernetes cluster", info(serverVersion.String()))
		}
	}

	// kubectl version
	output, err = o.getCommandOutput("", "kubectl", "version", "--short")
	if err != nil {
		o.warnf("Failed to get kubectl version: %s\n", err)
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
	args := []string{"version", "--short"}
	if o.HelmTLS {
		args = append(args, "--tls")
	}
	output, err = o.getCommandOutput("", "helm", args...)
	if err != nil {
		o.warnf("Failed to get helm version: %s\n", err)
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

	// git version
	output, err = o.getCommandOutput("", "git", "version")
	if err != nil {
		o.warnf("Failed to get git version: %s\n", err)
	} else {
		table.AddRow("git", info(output))
	}

	table.Render()

	if !o.NoVersionCheck {
		return o.versionCheck()
	}
	return nil
}

func (o *VersionOptions) versionCheck() error {
	newVersion, err := o.getLatestJXVersion()
	if err != nil {
		return err
	}

	currentVersion, err := version.GetSemverVersion()
	if err != nil {
		return err
	}

	if newVersion.GT(currentVersion) {
		app := util.ColorInfo("jx")
		o.Printf("\nA new %s version is available: %s\n", app, util.ColorInfo(newVersion.String()))

		if o.BatchMode {
			o.Printf("To upgrade to this new version use: %s\n", util.ColorInfo("jx upgrade cli"))
		} else {
			message := fmt.Sprintf("Would you like to upgrade to the new %s version?", app)
			if util.Confirm(message, true, "Please indicate if you would like to upgrade the binary version.") {
				return o.upgradeCli()
			}
		}
	}
	return nil
}

func (o *VersionOptions) upgradeCli() error {
	options := &UpgradeCLIOptions{
		CreateOptions: CreateOptions{
			CommonOptions: o.CommonOptions,
		},
	}
	return options.Run()
}

func extractSemVer(text string) string {
	re, err := regexp.Compile(".*SemVer:\"(.*)\"")
	if err != nil {
		return ""
	}
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
