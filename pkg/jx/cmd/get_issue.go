package cmd

import (
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
)

// GetIssueOptions contains the command line options
type GetIssueOptions struct {
	GetOptions

	Dir string
	Id  string
}

var (
	GetIssueLong = templates.LongDesc(`
		Display the status of an issue for a project.

`)

	GetIssueExample = templates.Examples(`
		# Get the status of an issue for a project
		jx get issue --id ISSUE_ID
	`)
)

// NewCmdGetIssue creates the command
func NewCmdGetIssue(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetIssueOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "issue [flags]",
		Short:   "Display the status of an issue",
		Long:    GetIssueLong,
		Example: GetIssueExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Id, "id", "i", "", "The issue ID")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The root project directory")

	options.addGetFlags(cmd)
	return cmd
}

// Run implements this command
func (o *GetIssueOptions) Run() error {
	tracker, err := o.createIssueProvider(o.Dir)
	if err != nil {
		return errors.Wrap(err, "failed to create the issue tracker")
	}

	issue, err := tracker.GetIssue(o.Id)
	if err != nil {
		return errors.Wrap(err, "issue not found")
	}

	table := o.CreateTable()
	table.AddRow("ISSUE", "STATUS", "ENVIRONMENT")
	if !issue.IsPullRequest {
		table.AddRow(issue.URL, *issue.State, "")
		table.Render()
		return nil
	}

	f := o.Factory
	client, ns, err := f.CreateJXClient()
	if err != nil {
		return errors.Wrap(err, "cannot create the JX client")
	}

	apisClient, err := f.CreateApiExtensionsClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the Kube API extensions client")
	}
	err = kube.RegisterEnvironmentCRD(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Environment API")
	}

	envList, err := client.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "cannot list the environments")
	}

	found := false
	for _, env := range envList.Items {
		envNs, err := kube.GetEnvironmentNamespace(client, ns, env.Name)
		if err != nil {
			continue
		}
		releaseList, err := client.JenkinsV1().Releases(envNs).List(metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "cannot list the releases")
		}
		rel := o.findRelease(issue.URL, releaseList.Items)
		if rel == nil {
			continue
		}
		table.AddRow(issue.URL, *issue.State, env.Name)
		found = true
	}
	if !found {
		table.AddRow(issue.URL, *issue.State, "")
	}
	table.Render()
	return nil
}

func (o *GetIssueOptions) findRelease(prURL string, releases []v1.Release) *v1.Release {
	for _, rel := range releases {
		prs := rel.Spec.PullRequests
		for _, pr := range prs {
			if pr.URL == prURL {
				return &rel
			}
		}
	}
	return nil
}
