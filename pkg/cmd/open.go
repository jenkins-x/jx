package cmd

import (
	"fmt"

	"github.com/jenkins-x/jx/v2/pkg/cmd/get"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
)

type OpenOptions struct {
	ConsoleOptions
}

type ConsoleOptions struct {
	get.GetURLOptions

	OnlyViewURL     bool
	ClassicMode     bool
	JenkinsSelector opts.JenkinsSelectorOptions
}

var (
	open_long = templates.LongDesc(`
		Opens a named service in the browser.

		You can use the '--url' argument to just display the URL without opening it`)

	open_example = templates.Examples(`
		# Open the Nexus console in a browser
		jx open jenkins-x-sonatype-nexus

		# Print the Nexus console URL but do not open a browser
		jx open jenkins-x-sonatype-nexus -u

		# List all the service URLs
		jx open`)
)

func NewCmdOpen(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &OpenOptions{
		ConsoleOptions: ConsoleOptions{
			GetURLOptions: get.GetURLOptions{
				Options: get.Options{
					CommonOptions: commonOpts,
				},
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "open",
		Short:   "Open a service in a browser",
		Long:    open_long,
		Example: open_example,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	options.addConsoleFlags(cmd)
	return cmd
}

func (o *OpenOptions) Run() error {
	if len(o.Args) == 0 {
		return o.GetURLOptions.Run()
	}
	name := o.Args[0]
	return o.ConsoleOptions.open(name, name)
}

func (o *ConsoleOptions) open(name string, label string) error {
	var err error
	svcURL := ""
	ns := o.Namespace
	if ns == "" && o.Environment != "" {
		ns, err = o.FindEnvironmentNamespace(o.Environment)
		if err != nil {
			return err
		}
	}
	if ns != "" {
		svcURL, err = o.FindServiceInNamespace(name, ns)
	} else {
		svcURL, err = o.FindService(name)
	}
	if err != nil && name != "" {
		log.Logger().Infof("If the app %s is running in a different environment you could try: %s", util.ColorInfo(name), util.ColorInfo("jx get applications"))
	}
	if err != nil {
		return err
	}
	fullURL := svcURL
	if name == "jenkins" {
		fullURL = o.urlForMode(svcURL)
	}
	if o.OnlyViewHost {
		host := util.URLToHostName(svcURL)
		_, err = fmt.Fprintf(o.Out, "%s: %s\n", label, util.ColorInfo(host))
		if err != nil {
			return err
		}
	} else {
		_, err = fmt.Fprintf(o.Out, "%s: %s\n", label, util.ColorInfo(fullURL))
		if err != nil {
			return err
		}
	}
	if !o.OnlyViewURL && !o.OnlyViewHost {
		err = browser.OpenURL(fullURL)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *ConsoleOptions) addConsoleFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&o.OnlyViewURL, "url", "u", false, "Only displays and the URL and does not open the browser")
	cmd.Flags().BoolVarP(&o.ClassicMode, "classic", "", false, "Use the classic Jenkins skin instead of Blue Ocean")

	o.AddGetUrlFlags(cmd)
	o.JenkinsSelector.AddFlags(cmd)
}

func (o *ConsoleOptions) urlForMode(url string) string {
	if o.ClassicMode {
		return url
	}

	blueOceanPath := "/blue"

	return util.UrlJoin(url, blueOceanPath)

}
