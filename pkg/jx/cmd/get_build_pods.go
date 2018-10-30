package cmd

import (
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"strings"
	"time"
)

// GetBuildPodsOptions the command line options
type GetBuildPodsOptions struct {
	GetOptions

	Namespace  string
	Owner      string
	Repository string
	Branch     string
	Build      string
	Filter     string
}

var (
	getBiuldPodsLong = templates.LongDesc(`
		Display the knative build pods

`)

	getBiuldPodsExample = templates.Examples(`
		# List all the knative build pods
		jx get build pods

		# List all the knative build pods for a given repository
		jx get build pods --repo cheese
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
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filters the build name by the given text")
	cmd.Flags().StringVarP(&options.Owner, "owner", "o", "", "Filters the owner (person/organisation) of the repository")
	cmd.Flags().StringVarP(&options.Repository, "repo", "r", "", "Filters the build repository")
	cmd.Flags().StringVarP(&options.Branch, "branch", "", "", "Filters the branch")
	cmd.Flags().StringVarP(&options.Build, "build", "b", "", "Filter a specific build number")
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

	table := o.CreateTable()
	table.AddRow("OWNER", "REPOSITORY", "BRANCH", "BUILD", "AGE", "STEP 1 IMAGE", "POD", "GIT URL")

	buildInfos := []*builds.BuildPodInfo{}
	for _, pod := range pods {
		buildInfo := builds.CreateBuildPodInfo(pod)
		if o.BuildMatches(buildInfo) {
			buildInfos = append(buildInfos, buildInfo)
		}
	}
	builds.SortBuildPodInfos(buildInfos)

	now := time.Now()
	for _, build := range buildInfos {
		duration := strings.TrimSuffix(now.Sub(build.CreatedTime).Round(time.Minute).String(), "0s")

		table.AddRow(build.Organisation, build.Repository, build.Branch, build.Build, duration, build.FirstStepImage, build.PodName, build.GitURL)
	}
	table.Render()
	return nil
}

func (o *GetBuildPodsOptions) BuildMatches(info *builds.BuildPodInfo) bool {
	if o.Owner != "" && o.Owner != info.Organisation {
		return false
	}
	if o.Repository != "" && o.Repository != info.Repository {
		return false
	}
	if o.Branch != "" && o.Branch != info.Branch {
		return false
	}
	if o.Build != "" && o.Build != info.Build {
		return false
	}
	if o.Filter != "" && !strings.Contains(info.Name, o.Filter) {
		return false
	}
	return true
}
