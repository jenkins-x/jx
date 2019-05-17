package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

var (
	getPostPreviewJobLong = templates.LongDesc(`
		Gets the jobs which are triggered after a Preview is created 
`)

	getPostPreviewJobExample = templates.Examples(`
		# List the jobs triggered after a Preview is created 
		jx get post preview job 

	`)
)

// GetPostPreviewJobOptions the options for the create spring command
type GetPostPreviewJobOptions struct {
	CreateOptions
}

// NewCmdGetPostPreviewJob creates a command object for the "create" command
func NewCmdGetPostPreviewJob(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetPostPreviewJobOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "post preview job",
		Short:   "Create a job which is triggered after a Preview is created",
		Aliases: branchPatternsAliases,
		Long:    getPostPreviewJobLong,
		Example: getPostPreviewJobExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	return cmd
}

// Run implements the command
func (o *GetPostPreviewJobOptions) Run() error {
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	table := o.CreateTable()
	table.AddRow("NAME", "IMAGE", "BACKOFF_LIMIT", "COMMAND")

	for _, job := range settings.PostPreviewJobs {
		name := job.Name
		image := ""
		commands := []string{}
		podSpec := &job.Spec.Template.Spec
		if len(podSpec.Containers) > 0 {
			container := &podSpec.Containers[0]
			image = container.Image
			commands = container.Command
		}
		backoffLimit := ""
		if job.Spec.BackoffLimit != nil {
			backoffLimit = strconv.Itoa(int(*job.Spec.BackoffLimit))
		}
		table.AddRow(name, image, backoffLimit, strings.Join(commands, " "))
	}
	table.Render()

	return nil
}
