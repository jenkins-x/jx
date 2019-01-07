package cmd

import (
	"io"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

// GetBuildPodsOptions the command line options
type GetBuildPodsOptions struct {
	GetOptions

	Namespace   string
	BuildFilter builds.BuildPodInfoFilter
}

var (
	getBiuldPodsLong = templates.LongDesc(`
		Display the knative build pods

`)

	getBiuldPodsExample = templates.Examples(`
		# List all the knative build pods
		jx get build pods

		# List all the pending knative build pods 
		jx get build pods -p

		# List all the knative build pods for a given repository
		jx get build pods --repo cheese

		# List all the pending knative build pods for a given repository
		jx get build pods --repo cheese -p

		# List all the knative build pods for a given Pull Request
		jx get build pods --repo cheese --branch PR-1234
	`)
)

// NewCmdGetBuildPods creates the command
func NewCmdGetBuildPods(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetBuildPodsOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,

				Out: out,
				Err: errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "pods [flags]",
		Short:   "Displays the build pods and their details",
		Long:    getBiuldPodsLong,
		Example: getBiuldPodsExample,
		Aliases: []string{"pod"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to look for the build pods. Defaults to the current namespace")
	cmd.Flags().BoolVarP(&options.BuildFilter.Pending, "pending", "p", false, "Filter builds which are currently pending or running")
	cmd.Flags().StringVarP(&options.BuildFilter.Filter, "filter", "f", "", "Filters the build name by the given text")
	cmd.Flags().StringVarP(&options.BuildFilter.Owner, "owner", "o", "", "Filters the owner (person/organisation) of the repository")
	cmd.Flags().StringVarP(&options.BuildFilter.Repository, "repo", "r", "", "Filters the build repository")
	cmd.Flags().StringVarP(&options.BuildFilter.Branch, "branch", "", "", "Filters the branch")
	cmd.Flags().StringVarP(&options.BuildFilter.Build, "build", "b", "", "Filter a specific build number")
	return cmd
}

// Run implements this command
func (o *GetBuildPodsOptions) Run() error {
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	if o.Namespace != "" {
		ns = o.Namespace
	}
	pods, err := builds.GetBuildPods(kubeClient, ns)
	if err != nil {
		log.Warnf("Failed to query pods %s\n", err)
		return err
	}

	table := o.createTable()
	table.AddRow("OWNER", "REPOSITORY", "BRANCH", "BUILD", "AGE", "STATUS", "STEP 1 IMAGE", "POD", "GIT URL")

	buildInfos := []*builds.BuildPodInfo{}
	for _, pod := range pods {
		buildInfo := builds.CreateBuildPodInfo(pod)
		if o.BuildFilter.BuildMatches(buildInfo) {
			buildInfos = append(buildInfos, buildInfo)
		}
	}
	builds.SortBuildPodInfos(buildInfos)

	now := time.Now()
	for _, build := range buildInfos {
		duration := strings.TrimSuffix(now.Sub(build.CreatedTime).Round(time.Minute).String(), "0s")

		table.AddRow(build.Organisation, build.Repository, build.Branch, build.Build, duration, build.Status(), build.FirstStepImage, build.PodName, build.GitURL)
	}
	table.Render()
	return nil
}
