package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/clients"
	"github.com/jenkins-x/jx/pkg/jx/cmd/commoncmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	optionImage = "image"
)

var (
	createPostPreviewJobLong = templates.LongDesc(`
		Create a job which is triggered after a Preview is created 
`)

	createPostPreviewJobExample = templates.Examples(`
		# Create a post preview job 
		jx create post preview job --name owasp --image owasp/zap2docker-stable:latest -c "zap-baseline.py" -c "-t" -c "\$(JX_PREVIEW_URL)" 

	`)
)

// CreatePostPreviewJobOptions the options for the create spring command
type CreatePostPreviewJobOptions struct {
	CreateOptions

	Name         string
	Image        string
	Commands     []string
	BackoffLimit int32
}

// NewCmdCreatePostPreviewJob creates a command object for the "create" command
func NewCmdCreatePostPreviewJob(f clients.Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreatePostPreviewJobOptions{
		CreateOptions: CreateOptions{
			CommonOptions: commoncmd.CommonOptions{
				Factory: f,
				In:      in,

				Out: out,
				Err: errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "post preview job",
		Short:   "Create a job which is triggered after a Preview is created",
		Aliases: branchPatternsAliases,
		Long:    createPostPreviewJobLong,
		Example: createPostPreviewJobExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Name, optionName, "n", "", "The name of the job")
	cmd.Flags().StringVarP(&options.Image, optionImage, "i", "", "The image to run in the jobb")
	cmd.Flags().StringArrayVarP(&options.Commands, "commands", "c", []string{}, "The commands to run in the job")
	cmd.Flags().Int32VarP(&options.BackoffLimit, "backoff-limit", "l", int32(2), "The backoff limit: how many times to retry the job before considering it failed) to run in the Job")

	options.AddCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreatePostPreviewJobOptions) Run() error {
	name := o.Name
	commands := o.Commands
	image := o.Image
	if name == "" {
		return util.MissingOption(optionName)
	}
	if image == "" {
		return util.MissingOption(optionImage)
	}
	labels := map[string]string{
		kube.LabelCreatedBy: kube.ValueCreatedByJX,
		kube.LabelJobKind:   kube.ValueJobKindPostPreview,
	}

	firstContainer := corev1.Container{
		Name:    name,
		Image:   image,
		Command: commands,
	}

	callback := func(env *v1.Environment) error {
		settings := &env.Spec.TeamSettings
		for i, _ := range settings.PostPreviewJobs {
			job := &settings.PostPreviewJobs[i]
			if job.Name == name {
				podSpec := &job.Spec.Template.Spec
				if len(podSpec.Containers) == 0 {
					podSpec.Containers = []corev1.Container{firstContainer}
				} else {
					container := &podSpec.Containers[0]
					container.Name = name
					container.Image = image
					container.Command = commands
				}
				job.Spec.BackoffLimit = &o.BackoffLimit
				log.Infof("Updating the post Preview Job: %s\n", util.ColorInfo(name))
				return nil
			}
		}
		settings.PostPreviewJobs = append(settings.PostPreviewJobs, batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: labels,
			},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers:    []corev1.Container{firstContainer},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
				BackoffLimit: &o.BackoffLimit,
			},
		})
		log.Infof("Added the post Preview Job: %s\n", util.ColorInfo(name))
		return nil
	}
	return o.ModifyDevEnvironment(callback)
}
