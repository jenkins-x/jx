package cloudbees

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/create"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/jenkins-x/jx/pkg/kube/services"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/browser"
)

type CloudBeesOptions struct {
	*opts.CommonOptions

	OnlyViewURL bool
}

var (
	core_long = templates.LongDesc(`
		Opens the CloudBees UI in a browser.

		Which helps you visualise your CI/CD pipelines, apps, environments and teams.

		For more information please see [https://www.cloudbees.com/blog/want-help-build-cloudbees-kubernetes-jenkins-x](https://www.cloudbees.com/blog/want-help-build-cloudbees-kubernetes-jenkins-x)
`)
	core_example = templates.Examples(`
		# Open the core dashboard in a browser
		jx cloudbees

		# Print the Jenkins X console URL but do not open a browser
		jx console -u`)
)

func NewCmdCloudBees(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CloudBeesOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "cloudbees",
		Short:   "Opens the CloudBees app for Kubernetes for visualising CI/CD and your environments",
		Long:    core_long,
		Example: core_example,
		Aliases: []string{"cloudbee", "cb", "ui", "jxui"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.AddCommand(NewCmdCloudBeesPipeline(commonOpts))
	cmd.Flags().BoolVarP(&options.OnlyViewURL, "url", "u", false, "Only displays and the URL and does not open the browser")
	return cmd
}

func (o *CloudBeesOptions) Run() error {
	url, err := o.GetBaseURL()
	if err != nil {
		return err
	}
	return o.OpenURL(url, "CloudBees")
}

func (o *CloudBeesOptions) GetBaseURL() (url string, err error) {
	client, err := o.KubeClient()
	if err != nil {
		return "", err
	}
	url, err = services.GetServiceURLFromName(client, kube.ServiceCloudBees, create.DefaultCloudBeesNamespace)
	if err != nil {
		return "", fmt.Errorf("%s\n\nDid you install the CloudBees addon via: %s\n\nFor more information see: %s", err, util.ColorInfo("jx create addon cloudbees"), util.ColorInfo("https://www.cloudbees.com/blog/want-help-build-cloudbees-kubernetes-jenkins-x"))
	}

	if url == "" {
		url, err = services.GetServiceURLFromName(client, fmt.Sprintf("sso-%s", kube.ServiceCloudBees), create.DefaultCloudBeesNamespace)
		if err != nil {
			return "", fmt.Errorf("%s\n\nDid you install the CloudBees addon via: %s\n\nFor more information see: %s", err, util.ColorInfo("jx create addon cloudbees"), util.ColorInfo("https://www.cloudbees.com/blog/want-help-build-cloudbees-kubernetes-jenkins-x"))
		}
	}
	return url, nil
}

func (o *CloudBeesOptions) Open(name string, label string) error {
	url, err := o.FindService(name)
	if err != nil {
		return err
	}
	return o.OpenURL(url, label)
}

func (o *CloudBeesOptions) OpenURL(url string, label string) error {
	// TODO Logger
	fmt.Fprintf(o.Out, "%s: %s\n", label, util.ColorInfo(url))
	if !o.OnlyViewURL {
		browser.OpenURL(url)
	}
	return nil
}
