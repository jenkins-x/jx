package cmd

import (
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
)

var (
	deletePostPreviewJobLong = templates.LongDesc(`
		Delete a job which is triggered after a Preview is created 
`)

	deletePostPreviewJobExample = templates.Examples(`
		# Delete a post preview job 
		jx delete post preview job --name owasp 

	`)
)

// DeletePostPreviewJobOptions the options for the create spring command
type DeletePostPreviewJobOptions struct {
	DeleteOptions

	Name string
}

// NewCmdDeletePostPreviewJob creates a command object for the "create" command
func NewCmdDeletePostPreviewJob(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &DeletePostPreviewJobOptions{
		DeleteOptions: DeleteOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "post preview job",
		Short:   "Create a job which is triggered after a Preview is created",
		Aliases: branchPatternsAliases,
		Long:    deletePostPreviewJobLong,
		Example: deletePostPreviewJobExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Name, optionName, "n", "", "The name of the job to be deleted")
	return cmd
}

// Run implements the command
func (o *DeletePostPreviewJobOptions) Run() error {
	name := o.Name
	if name == "" {
		// TODO if not batch mode then lets let the user pick?
		return util.MissingOption(optionName)
	}

	callback := func(env *v1.Environment) error {
		settings := &env.Spec.TeamSettings
		idx := -1
		for i, job := range settings.PostPreviewJobs {
			if job.Name == name {
				idx = i
				break
			}
		}
		if idx >= 0 {
			settings.PostPreviewJobs = append(settings.PostPreviewJobs[0:idx], settings.PostPreviewJobs[idx+1:]...)
			logrus.Infof("Deleting the post Preview Job: %s\n", util.ColorInfo(name))
		} else {
			logrus.Warnf("post Preview Job: %s does not exist in this team\n", name)
		}
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
