package cmd

import (
	"io"

	"github.com/jenkins-x/golang-jenkins"
	"github.com/spf13/cobra"

	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/util"
)

// GetBuildLogsOptions the command line options
type GetBuildLogsOptions struct {
	GetOptions

	Tail   bool
	Filter string
	Build  int
}

var (
	get_build_log_long = templates.LongDesc(`
		Display the git server URLs.

`)

	get_build_log_example = templates.Examples(`
		# List all registered git server URLs
		jx get git
	`)
)

// NewCmdGetBuildLogs creates the command
func NewCmdGetBuildLogs(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &GetBuildLogsOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "log [flags]",
		Short:   "Display a build log",
		Long:    get_build_log_long,
		Example: get_build_log_example,
		Aliases: []string{"logs"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Tail, "tail", "t", true, "Tails the build log to the current terminal")
	cmd.Flags().StringVarP(&options.Filter, "filter", "f", "", "Filters all the available jobs by those that contain the given text")
	cmd.Flags().IntVarP(&options.Build, "build", "b", 0, "The build number to view")

	return cmd
}

// Run implements this command
func (o *GetBuildLogsOptions) Run() error {
	jobMap, err := o.getJobMap(o.Filter)
	if err != nil {
		return err
	}
	jenkinsClient, err := o.JenkinsClient()
	if err != nil {
		return err
	}

	args := o.Args
	names := []string{}
	for k, _ := range jobMap {
		names = append(names, k)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("No pipelines have been built!")
	}

	if len(args) == 0 {
		defaultName := ""
		for _, n := range names {
			if strings.HasSuffix(n, "/master") {
				defaultName = n
				break
			}
		}
		name, err := util.PickNameWithDefault(names, "Which pipeline do you want to view the logs of?: ", defaultName)
		if err != nil {
			return err
		}
		args = []string{name}
	}
	if len(args) == 0 {
		return fmt.Errorf("No pipeline chosen")
	}
	name := args[0]
	job := jobMap[name]
	var last gojenkins.Build
	if o.Build > 0 {
		last, err = jenkinsClient.GetBuild(job, o.Build)
	} else {
		last, err = jenkinsClient.GetLastBuild(job)
	}
	if err != nil {
		return err
	}
	o.Printf("%s %s\n", util.ColorStatus("view the log at:"), util.ColorInfo(util.UrlJoin(last.Url, "/console")))
	return o.tailBuild(name, &last)
}
