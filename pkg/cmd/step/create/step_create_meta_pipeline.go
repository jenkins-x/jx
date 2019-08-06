package create

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	contextOptionName        = "context"
	defaultImageOptionName   = "default-image"
	envOptionName            = "env"
	labelOptionName          = "label"
	noApplyOptionName        = "no-apply"
	outputOptionName         = "output"
	pullRefOptionName        = "pull-refs"
	serviceAccountOptionName = "service-account"
	sourceURLOptionName      = "source-url"
)

var (
	createPipelineLong = templates.LongDesc(`
		Creates and applies the meta pipeline Tekton CRDs allowing apps to extend the build pipeline.
`)

	createPipelineExample = templates.Examples(`
		# Create the Tekton meta pipeline which allows Jenkins-X Apps to extend the actual build pipeline.
		jx step create pipeline
			`)

	createPipelineOutDir  string
	createPipelineNoApply bool
)

// StepCreatePipelineOptions contains the command line flags for the command to create the meta pipeline
type StepCreatePipelineOptions struct {
	*opts.CommonOptions

	Client  metapipeline.Client
	Results tekton.CRDWrapper
	NoApply *bool
}

// NewCmdCreateMetaPipeline creates the command for generating and applying the Tekton CRDs for the meta pipeline.
func NewCmdCreateMetaPipeline(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepCreatePipelineOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "pipeline",
		Short:   "Creates the Tekton meta pipeline for a given build pipeline.",
		Long:    createPipelineLong,
		Example: createPipelineExample,
		Aliases: []string{"bt"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
		Hidden: true,
	}

	cmd.Flags().StringVar(&options.Client.SourceURL, sourceURLOptionName, "", "Specify the git URL for the source (required)")

	cmd.Flags().StringVar(&options.Client.PullRefs, pullRefOptionName, "", "The Prow pull ref specifying the references to merge into the source")
	cmd.Flags().StringVarP(&options.Client.Context, contextOptionName, "c", "", "The pipeline context if there are multiple separate pipelines for a given branch")

	cmd.Flags().StringVar(&options.Client.DefaultImage, defaultImageOptionName, syntax.DefaultContainerImage, "Specify the docker image to use if there is no image specified for a step. Default "+syntax.DefaultContainerImage)
	cmd.Flags().StringArrayVarP(&options.Client.CustomLabels, labelOptionName, "l", nil, "List of custom labels to be applied to the generated PipelineRun (can be use multiple times)")
	cmd.Flags().StringArrayVarP(&options.Client.CustomEnvs, envOptionName, "e", nil, "List of custom environment variables to be applied to resources that are created (can be use multiple times)")

	cmd.Flags().StringVar(&options.Client.ServiceAccount, serviceAccountOptionName, "tekton-bot", "The Kubernetes ServiceAccount to use to run the pipeline")

	// options to control the output, mainly for development
	cmd.Flags().BoolVar(&createPipelineNoApply, noApplyOptionName, false, "Disables creating the pipeline resources in the cluster and just outputs the generated resources to file")
	cmd.Flags().StringVarP(&createPipelineOutDir, outputOptionName, "o", "out", "Used in conjunction with --no-apply to determine the directory into which to write the output")

	options.AddCommonFlags(cmd)
	options.setupViper(cmd)
	return cmd
}

func (o *StepCreatePipelineOptions) setupViper(cmd *cobra.Command) {
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	_ = viper.BindEnv(noApplyOptionName)
	_ = viper.BindPFlag(noApplyOptionName, cmd.Flags().Lookup(noApplyOptionName))

	_ = viper.BindEnv(outputOptionName)
	_ = viper.BindPFlag(outputOptionName, cmd.Flags().Lookup(outputOptionName))
}

// Run implements this command
func (o *StepCreatePipelineOptions) Run() error {
	if o.NoApply == nil {
		b := viper.GetBool(noApplyOptionName)
		o.NoApply = &b
	}
	o.Client.NoApply = *o.NoApply

	if o.Client.OutDir == "" {
		s := viper.GetString(outputOptionName)
		o.Client.OutDir = s
	}

	err := o.Client.Create()
	if err != nil {
		return errors.Wrapf(err, "failed to create Meta Pipeline")
	}
	// record the results in the struct for the case this command is called programmatically (HF)
	o.Results = o.Client.Results
	return nil
}
