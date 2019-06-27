package create

import (
	"fmt"
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
	"strings"
	"time"

	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	kubeclient "k8s.io/client-go/kubernetes"
)

const (
	branchOptionName         = "branch"
	contextOptionName        = "context"
	defaultImageOptionName   = "default-image"
	envOptionName            = "env"
	jobOptionName            = "job"
	labelOptionName          = "label"
	noApplyOptionName        = "no-apply"
	outputOptionName         = "output"
	pipelineKindOptionName   = "kind"
	pullRefOptionName        = "pull-refs"
	pullRequestOptionName    = "pr-number"
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
)

// StepCreatePipelineOptions contains the command line flags for the command to create the meta pipeline
type StepCreatePipelineOptions struct {
	*opts.CommonOptions

	SourceURL    string
	Branch       string
	PullRefs     string
	PipelineKind string
	Job          string

	PullRequestNumber string
	Context           string

	CustomLabels []string
	CustomEnvs   []string
	DefaultImage string

	OutDir  string
	NoApply bool
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
	cmd.Flags().StringVar(&options.Branch, branchOptionName, "", "The git branch to trigger the build in. Defaults to the current local branch name (required)")
	cmd.Flags().StringVarP(&options.PipelineKind, pipelineKindOptionName, "k", "", "The kind of pipeline to create such as: "+strings.Join(jenkinsfile.PipelineKinds, ", "))
	cmd.Flags().StringVar(&options.Job, jobOptionName, "", "The Prow job name in order to identify all pipelines belonging to a single trigger event (required)")

	cmd.Flags().StringVar(&options.PullRefs, pullRefOptionName, "", "The Prow pull ref specifying the references to merge into the source")
	cmd.Flags().StringVarP(&options.Context, contextOptionName, "c", "", "The pipeline context if there are multiple separate pipelines for a given branch")

	cmd.Flags().StringVar(&options.DefaultImage, defaultImageOptionName, syntax.DefaultContainerImage, "Specify the docker image to use if there is no image specified for a step. Default "+syntax.DefaultContainerImage)
	cmd.Flags().StringArrayVarP(&options.CustomLabels, labelOptionName, "l", nil, "List of custom labels to be applied to the generated PipelineRun (can be use multiple times)")
	cmd.Flags().StringArrayVarP(&options.CustomEnvs, envOptionName, "e", nil, "List of custom environment variables to be applied to resources that are created (can be use multiple times)")

	cmd.Flags().StringVar(&options.PullRequestNumber, pullRequestOptionName, "", "If a pull request this is it's number")
	cmd.Flags().StringVar(&options.ServiceAccount, serviceAccountOptionName, "tekton-bot", "The Kubernetes ServiceAccount to use to run the pipeline")

	// options to control the output, mainly for development
	cmd.Flags().BoolVar(&options.NoApply, noApplyOptionName, false, "Disables creating the pipeline resources in the cluster and just outputs the generated resources to file")
	cmd.Flags().StringVarP(&options.OutDir, outputOptionName, "o", "out", "Used in conjunction with --no-apply to determine the directory into which to write the output")

	options.AddCommonFlags(cmd)
	return cmd
}

// Run implements this command
func (o *StepCreatePipelineOptions) Run() error {
	err := o.validateCommandLineFlags()
	if err != nil {
		return err
	}

	gitInfo, _ := gits.ParseGitURL(o.SourceURL)

	tektonClient, jxClient, kubeClient, ns, err := o.getClientsAndNamespace()
	if err != nil {
		return err
	}

	podTemplates, err := o.getPodTemplates(kubeClient, ns, apps.AppPodTemplateName)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve pod templates")
	}

	pipelineName := tekton.PipelineResourceNameFromGitInfo(gitInfo, o.Branch, o.Context, tekton.MetaPipeline)
	buildNumber, err := tekton.GenerateNextBuildNumber(tektonClient, jxClient, ns, gitInfo, o.Branch, retryDuration, pipelineName)
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

	crdCreationParams := metapipeline.CRDCreationParameters{
		Namespace:      ns,
		Context:        o.Context,
		PipelineName:   pipelineName,
		PipelineKind:   o.PipelineKind,
		BuildNumber:    buildNumber,
		GitInfo:        gitInfo,
		Branch:         o.Branch,
		PullRef:        o.PullRefs,
		SourceDir:      defaultCheckoutDir,
		PodTemplates:   podTemplates,
		Trigger:        string(pipelineapi.PipelineTriggerTypeManual),
		ServiceAccount: o.ServiceAccount,
		Labels:         o.CustomLabels,
		EnvVars:        o.CustomEnvs,
		DefaultImage:   o.DefaultImage,
		Apps:           extendingApps,
	}
	tektonCRDs, err := metapipeline.CreateMetaPipelineCRDs(crdCreationParams)
	if err != nil {
		return errors.Wrap(err, "failed to generate Tekton CRDs for meta pipeline")
	}

	err = o.handleResult(tektonClient, jxClient, tektonCRDs, buildNumber, o.Branch, o.PullRefs, ns, gitInfo)
	if err != nil {
		return err
	}

	return nil
}

func (o *StepCreatePipelineOptions) validateCommandLineFlags() error {
	if o.SourceURL == "" {
		return util.MissingOption(sourceURLOptionName)
	}

	_, err := gits.ParseGitURL(o.SourceURL)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to determine needed git info from the specified git url '%s'", o.SourceURL))
	}

	if o.Branch == "" {
		return util.MissingOption(branchOptionName)
	}

	if o.PipelineKind == "" {
		return util.MissingOption(pipelineKindOptionName)
	}

	if o.Job == "" {
		return util.MissingOption(jobOptionName)
	}

	if util.StringArrayIndex(jenkinsfile.PipelineKinds, o.PipelineKind) == -1 {
		return util.InvalidOption(pipelineKindOptionName, o.PipelineKind, jenkinsfile.PipelineKinds)
	}

	_, err = util.ExtractKeyValuePairs(o.CustomLabels, "=")
	if err != nil {
		return errors.Wrap(err, "unable to parse custom labels")
	}

	_, err = util.ExtractKeyValuePairs(o.CustomEnvs, "=")
	if err != nil {
		return errors.Wrap(err, "unable to parse custom environment variables")
	}

	if o.PullRefs != "" {
		_, err = prow.ParsePullRefs(o.PullRefs)
		if err != nil {
			return errors.Wrapf(err, "unable to parse pull ref '%s'", o.PullRefs)
		}
	}

	return nil
}

func (o *StepCreatePipelineOptions) handleResult(tektonClient tektonclient.Interface,
	jxClient jxclient.Interface,
	tektonCRDs *tekton.CRDWrapper,
	buildNumber string,
	branch string,
	pullRefs string,
	ns string,
	gitInfo *gits.GitRepository) error {

	pr, err := o.buildPullRevs(pullRefs)
	if err != nil {
		return errors.Wrapf(err, "failed to build pull refs")
	}
	pipelineActivity := tekton.GeneratePipelineActivity(buildNumber, branch, gitInfo, pr, tekton.MetaPipeline)
	if o.NoApply {
		err := tektonCRDs.WriteToDisk(o.OutDir, pipelineActivity)
		if err != nil {
			return errors.Wrapf(err, "failed to output Tekton CRDs")
		}
	} else {
		err := tekton.ApplyPipeline(jxClient, tektonClient, tektonCRDs, ns, gitInfo, o.Branch, pipelineActivity)
		if err != nil {
			return errors.Wrapf(err, "failed to apply Tekton CRDs")
		}
		if o.Verbose {
			log.Logger().Infof("applied tekton CRDs for %s\n", tektonCRDs.PipelineRun().Name)
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

func (o *StepCreatePipelineOptions) buildPullRevs(pullRefs string) (*prow.PullRefs, error) {
	var pr *prow.PullRefs
	var err error
	if pullRefs != "" {
		pr, err = prow.ParsePullRefs(pullRefs)
	}
	return pr, err
}
