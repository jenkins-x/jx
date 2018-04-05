package cmd

import (
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

	Container string
	Namespace string
	HelmTLS   bool
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
	cmd.Flags().MarkShorthandDeprecated("client", "please use --client instead.")
	cmd.Flags().BoolVarP(&options.HelmTLS, "helm-tls", "", false, "Whether to use TLS with helm")
	return cmd
}

func (o *VersionOptions) Run() error {
	info := util.ColorInfo
	table := o.CreateTable()
	table.AddRow("NAME", "VERSION")
	table.AddRow("jx", info(version.GetVersion()))

	// Jenkins X version
	output, err := o.getCommandOutput("", "helm", "list")
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
						table.AddRow("Jenkins X", info(chart))
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
			table.AddRow("Kubernetes", info(serverVersion.String()))
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
						table.AddRow("Helm Client", info(v))
					case 1:
						table.AddRow("Helm Server", info(v))
					}
				}
			}
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
						table.AddRow("Kubectl Client", info(v))
					case 1:
						// Ignore K8S server details as we have these above
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
		table.AddRow("Git", info(output))
	}

	table.Render()
	return nil
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
