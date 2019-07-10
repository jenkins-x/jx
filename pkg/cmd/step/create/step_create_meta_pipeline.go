package create

import (
	"fmt"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	kubeclient "k8s.io/client-go/kubernetes"
)

const (
	contextOptionName        = "context"
	defaultImageOptionName   = "default-image"
	envOptionName            = "env"
	jobOptionName            = "job"
	labelOptionName          = "label"
	noApplyOptionName        = "no-apply"
	outputOptionName         = "output"
	pullRefOptionName        = "pull-refs"
	serviceAccountOptionName = "service-account"
	sourceURLOptionName      = "source-url"

	retryDuration      = time.Second * 30
	defaultCheckoutDir = "source"
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

	SourceURL string
	PullRefs  string
	Context   string
	Job       string

	CustomLabels []string
	CustomEnvs   []string
	DefaultImage string

	Results tekton.CRDWrapper
	OutDir  string
	NoApply *bool

	VersionResolver *opts.VersionResolver
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

	cmd.Flags().StringVar(&options.SourceURL, sourceURLOptionName, "", "Specify the git URL for the source (required)")
	cmd.Flags().StringVar(&options.Job, jobOptionName, "", "The Prow job name in order to identify all pipelines belonging to a single trigger event (required)")

	cmd.Flags().StringVar(&options.PullRefs, pullRefOptionName, "", "The Prow pull ref specifying the references to merge into the source")
	cmd.Flags().StringVarP(&options.Context, contextOptionName, "c", "", "The pipeline context if there are multiple separate pipelines for a given branch")

	cmd.Flags().StringVar(&options.DefaultImage, defaultImageOptionName, syntax.DefaultContainerImage, "Specify the docker image to use if there is no image specified for a step. Default "+syntax.DefaultContainerImage)
	cmd.Flags().StringArrayVarP(&options.CustomLabels, labelOptionName, "l", nil, "List of custom labels to be applied to the generated PipelineRun (can be use multiple times)")
	cmd.Flags().StringArrayVarP(&options.CustomEnvs, envOptionName, "e", nil, "List of custom environment variables to be applied to resources that are created (can be use multiple times)")

	cmd.Flags().StringVar(&options.ServiceAccount, serviceAccountOptionName, "tekton-bot", "The Kubernetes ServiceAccount to use to run the pipeline")

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

	if o.OutDir == "" {
		s := viper.GetString(outputOptionName)
		o.OutDir = s
	}

	err := o.validateCommandLineFlags()
	if err != nil {
		return err
	}

	gitInfo, err := gits.ParseGitURL(o.SourceURL)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to determine needed git info from the specified git url '%s'", o.SourceURL))
	}

	pullRefs, err := prow.ParsePullRefs(o.PullRefs)
	if err != nil {
		return errors.Wrapf(err, "unable to parse pull ref '%s'", o.PullRefs)
	}

	tektonClient, jxClient, kubeClient, ns, err := o.getClientsAndNamespace()
	if err != nil {
		return err
	}

	podTemplates, err := o.getPodTemplates(kubeClient, ns, apps.AppPodTemplateName)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve pod templates")
	}

	branchIdentifier := o.determineBranchIdentifier(*pullRefs)
	pipelineKind := o.determinePipelineKind(*pullRefs)

	pipelineName := tekton.PipelineResourceNameFromGitInfo(gitInfo, branchIdentifier, o.Context, tekton.MetaPipeline, tektonClient, ns)
	buildNumber, err := tekton.GenerateNextBuildNumber(tektonClient, jxClient, ns, gitInfo, branchIdentifier, retryDuration, o.Context)
	if err != nil {
		return errors.Wrap(err, "unable to determine next build number")
	}

	if o.Verbose {
		log.Logger().Infof("creating meta pipeline CRDs for build %s of repository '%s/%s'", buildNumber, gitInfo.Organisation, gitInfo.Name)
	}
	extendingApps, err := metapipeline.GetExtendingApps(jxClient, ns)
	if err != nil {
		return err
	}

	if o.VersionResolver == nil {
		o.VersionResolver, err = o.CreateVersionResolver("", "")
		if err != nil {
			return err
		}
	}

	crdCreationParams := metapipeline.CRDCreationParameters{
		Namespace:        ns,
		Context:          o.Context,
		PipelineName:     pipelineName,
		PipelineKind:     pipelineKind,
		BuildNumber:      buildNumber,
		GitInfo:          *gitInfo,
		BranchIdentifier: branchIdentifier,
		PullRef:          *pullRefs,
		SourceDir:        defaultCheckoutDir,
		PodTemplates:     podTemplates,
		ServiceAccount:   o.ServiceAccount,
		Labels:           o.CustomLabels,
		EnvVars:          o.CustomEnvs,
		DefaultImage:     o.DefaultImage,
		Apps:             extendingApps,
		VersionsDir:      o.VersionResolver.VersionsDir,
	}
	tektonCRDs, err := metapipeline.CreateMetaPipelineCRDs(crdCreationParams)
	if err != nil {
		return errors.Wrap(err, "failed to generate Tekton CRDs for meta pipeline")
	}
	// record the results in the struct for the case this command is called programmatically (HF)
	o.Results = *tektonCRDs

	err = o.handleResult(tektonClient, jxClient, tektonCRDs, buildNumber, branchIdentifier, *pullRefs, ns, gitInfo)
	if err != nil {
		return err
	}

	return nil
}

func (o *StepCreatePipelineOptions) validateCommandLineFlags() error {
	if o.SourceURL == "" {
		return util.MissingOption(sourceURLOptionName)
	}

	if o.Job == "" {
		return util.MissingOption(jobOptionName)
	}

	_, err := util.ExtractKeyValuePairs(o.CustomLabels, "=")
	if err != nil {
		return errors.Wrap(err, "unable to parse custom labels")
	}

	_, err = util.ExtractKeyValuePairs(o.CustomEnvs, "=")
	if err != nil {
		return errors.Wrap(err, "unable to parse custom environment variables")
	}

	if o.PullRefs == "" {
		return util.MissingOption(pullRefOptionName)
	}

	return nil
}

func (o *StepCreatePipelineOptions) handleResult(tektonClient tektonclient.Interface,
	jxClient jxclient.Interface,
	tektonCRDs *tekton.CRDWrapper,
	buildNumber string,
	branch string,
	pullRefs prow.PullRefs,
	ns string,
	gitInfo *gits.GitRepository) error {

	pipelineActivity := tekton.GeneratePipelineActivity(buildNumber, branch, gitInfo, &pullRefs, tekton.MetaPipeline)
	if *o.NoApply {
		err := tektonCRDs.WriteToDisk(o.OutDir, pipelineActivity)
		if err != nil {
			return errors.Wrapf(err, "failed to output Tekton CRDs")
		}
	} else {
		err := tekton.ApplyPipeline(jxClient, tektonClient, tektonCRDs, ns, gitInfo, branch, pipelineActivity)
		if err != nil {
			return errors.Wrapf(err, "failed to apply Tekton CRDs")
		}
		if o.Verbose {
			log.Logger().Infof("applied tekton CRDs for %s", tektonCRDs.PipelineRun().Name)
		}
	}
	return nil
}

func (o *StepCreatePipelineOptions) getPodTemplates(kubeClient kubeclient.Interface, ns string, containerName string) (map[string]*corev1.Pod, error) {
	podTemplates, err := kube.LoadPodTemplates(kubeClient, ns)
	if err != nil {
		return nil, err
	}

	return podTemplates, nil
}

func (o *StepCreatePipelineOptions) getClientsAndNamespace() (tektonclient.Interface, jxclient.Interface, kubeclient.Interface, string, error) {
	tektonClient, _, err := o.TektonClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create Tekton client")
	}

	jxClient, _, err := o.JXClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create JX client")
	}

	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create Kube client")
	}

	return tektonClient, jxClient, kubeClient, ns, nil
}

// determineBranchIdentifier finds a identifier for the branch which is build by the specified pull ref.
// The name is either an actual git branch name (eg master) or a synthetic names  like PR-<number> for single pull requests
// or 'batch' for Prow batch builds.
// The method makes the decision purely based on the Prow pull ref. At this stage we don't have the full ProwJov spec
// available.
func (o *StepCreatePipelineOptions) determineBranchIdentifier(pullRef prow.PullRefs) string {
	prCount := len(pullRef.ToMerge)
	var branch string
	switch prCount {
	case 0:
		{
			// no pull requests to merge, taking base branch name as identifier
			branch = pullRef.BaseBranch
		}
	case 1:
		{
			// single pull request, create synthetic PR identifier
			for k := range pullRef.ToMerge {
				branch = fmt.Sprintf("PR-%s", k)
				break
			}
		}
	default:
		{
			branch = "batch"
		}
	}
	log.Logger().Debugf("branch identifier for pull ref '%s' : '%s'", pullRef.String(), branch)
	return branch
}

func (o *StepCreatePipelineOptions) determinePipelineKind(pullRef prow.PullRefs) string {
	var kind string

	prCount := len(pullRef.ToMerge)
	if prCount > 0 {
		kind = jenkinsfile.PipelineKindPullRequest
	} else {
		kind = jenkinsfile.PipelineKindRelease
	}
	log.Logger().Debugf("pipeline kind for pull ref '%s' : '%s'", pullRef.String(), kind)
	return kind
}
