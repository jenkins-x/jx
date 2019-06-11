package cloudbees

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/browser"
)

type CloudBeesPipelineOptions struct {
	CloudBeesOptions
}

var (
	cloudbees_pipeline_long = templates.LongDesc(`
		Opens the CloudBees Pipeline page for the current App in a browser.

		Which helps you visualise your CI/CD pipelines, apps, environments and teams.

		For more information please see [https://www.cloudbees.com/blog/want-help-build-cloudbees-kubernetes-jenkins-x](https://www.cloudbees.com/blog/want-help-build-cloudbees-kubernetes-jenkins-x)
`)
	cloudbees_pipeline_example = templates.Examples(`
		# Open the CloudBees Pipeline page in a browser
		jx cloudbees pipeline

		# Print the CloudBees Pipeline page URL but do not open a browser
		jx cloudbees pipeline -u`)
)

func NewCmdCloudBeesPipeline(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CloudBeesPipelineOptions{
		CloudBeesOptions: CloudBeesOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "pipeline",
		Short:   "Opens the CloudBees Pipeline page for visualising CI/CD",
		Long:    cloudbees_pipeline_long,
		Example: cloudbees_pipeline_example,
		Aliases: []string{"cloudbee", "cb", "core"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.OnlyViewURL, "url", "u", false, "Only displays the URL and does not open the browser")
	return cmd
}

func (o *CloudBeesPipelineOptions) Run() error {

	app, err := o.DiscoverAppName()
	if err != nil {
		return err
	}

	team, _, err := o.TeamAndEnvironmentNames()
	if err != nil {
		return err
	}

	baseUrl, err := o.GetBaseURL()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/teams/%s/pipelines/%s", baseUrl, team, app)

	return o.OpenURL(url, fmt.Sprintf("CloudBees (team: %s, app: %s)", util.ColorInfo(team), util.ColorInfo(app)))
}

func (o *CloudBeesPipelineOptions) Open(name string, label string) error {
	url, err := o.FindService(name)
	if err != nil {
		return err
	}
	return o.OpenURL(url, label)
}

func (o *CloudBeesPipelineOptions) OpenURL(url string, label string) error {
	// TODO Logger
	fmt.Fprintf(o.Out, "%s: %s\n", label, util.ColorInfo(url))
	if !o.OnlyViewURL {
		browser.OpenURL(url)
	}
	return nil
}
