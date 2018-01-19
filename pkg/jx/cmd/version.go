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
		return err
	}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 4 && fields[0] == "jenkins-x" {
			for _, f := range fields[4:] {
				if strings.HasPrefix(f, jxChartPrefix) {
					chart := strings.TrimPrefix(f, jxChartPrefix)
					table.AddRow("Jenkins X", info(chart))
				}
			}
		}
	}

	// kubernetes version
	client, _, err := o.KubeClient()
	if err != nil {
		return err
	}
	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	if serverVersion != nil {
		table.AddRow("Kubernetes", info(serverVersion.String()))
	}

	// helm version
	output, err = o.getCommandOutput("", "helm", "version", "--short")
	if err != nil {
		return err
	}
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

	// draft version
	output, err = o.getCommandOutput("", "draft", "version")
	if err != nil {
		return err
	}
	for i, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 {
			v := extractSemVer(fields[1])
			if v != "" {
				switch i {
				case 0:
					table.AddRow("Draft Client", info(v))
				case 1:
					table.AddRow("Draft Server", info(v))
				}
			}
		}
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
