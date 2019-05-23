package create

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
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
	gitCloneURLOptionName  = "clone-git-url"
	branchOptionName       = "branch"
	pullRefOptionName      = "pull-refs"
	pipelineKindOptionName = "kind"
	retryDuration          = time.Second * 30
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

	GitCloneURL  string
	Branch       string
	PullRefs     string
	PipelineKind string

	Revision          string
	PullRequestNumber string
	Context           string
	SourceName        string
	Trigger           string

	CustomLabels []string
	CustomEnvs   []string

	OutDir    string
	NoApply   bool
	ViewSteps bool
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

	cmd.Flags().StringVarP(&options.GitCloneURL, gitCloneURLOptionName, "", "", "Specify the git URL for the source (required)")
	cmd.Flags().StringVarP(&options.Branch, branchOptionName, "", "", "The git branch to trigger the build in. Defaults to the current local branch name")
	cmd.Flags().StringVarP(&options.PullRefs, pullRefOptionName, "", "", "The Prow pull ref specifying the references to merge into the source")
	cmd.Flags().StringVarP(&options.PipelineKind, pipelineKindOptionName, "k", "", "The kind of pipeline to create such as: "+strings.Join(jenkinsfile.PipelineKinds, ", "))

	cmd.Flags().StringArrayVarP(&options.CustomLabels, "label", "l", nil, "List of custom labels to be applied to the generated PipelineRun (can be use multiple times)")
	cmd.Flags().StringArrayVarP(&options.CustomEnvs, "env", "e", nil, "List of custom environment variables to be applied to resources that are created (can be use multiple times)")

	cmd.Flags().StringVarP(&options.Revision, "revision", "", "", "The git revision to checkout, can be a branch name or git sha")
	cmd.Flags().StringVarP(&options.PullRequestNumber, "pr-number", "", "", "If a Pull Request this is it's number")
	cmd.Flags().StringVarP(&options.SourceName, "source", "", "source", "The name under which to checkout the repository. Defaults to 'source'")
	cmd.Flags().StringVarP(&options.Trigger, "trigger", "t", string(pipelineapi.PipelineTriggerTypeManual), "The kind of pipeline trigger")
	cmd.Flags().StringVarP(&options.ServiceAccount, "service-account", "", "tekton-bot", "The Kubernetes ServiceAccount to use to run the pipeline")

	cmd.Flags().StringVarP(&options.OutDir, "output", "o", "out", "The directory to write the output to as YAML")
	cmd.Flags().BoolVarP(&options.NoApply, "no-apply", "", false, "Disables creating the pipeline resources in the kubernetes cluster and just outputs the generated Task to the console or output file")
	cmd.Flags().BoolVarP(&options.ViewSteps, "view", "", false, "Just view the steps that would be created")

	options.AddCommonFlags(cmd)
	return cmd
}

// Run implements this command
func (o *StepCreatePipelineOptions) Run() error {
	err := o.validateCommandLineFlags()
	if err != nil {
		return err
	}

	gitInfo, _ := gits.ParseGitURL(o.GitCloneURL)

	tektonClient, jxClient, kubeClient, ns, err := o.getClientsAndNamespace()
	if err != nil {
		return err
	}

	podTemplates, err := o.getPodTemplates(kubeClient, ns, apps.AppPodTemplateName)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve pod templates")
	}

	pipelineResourceName := tekton.PipelineResourceName(gitInfo, o.Branch, o.Context)
	buildNumber, err := tekton.GenerateNextBuildNumber(tektonClient, jxClient, ns, gitInfo, o.Branch, retryDuration, pipelineResourceName)
	if err != nil {
		return errors.Wrap(err, "unable to determine next build number")
	}

	if o.Verbose {
		log.Infof("creating meta pipeline CRDs for build %s of repository '%s/%s'", buildNumber, gitInfo.Organisation, gitInfo.Name)
	}
	extendingApps, err := metapipeline.GetExtendingApps(jxClient, ns)
	if err != nil {
		return err
	}

	pipelineName := tekton.PipelineResourceName(gitInfo, o.Branch, o.Context)
	labels := o.buildLabels(gitInfo)
	envVars := o.buildEnvVars()
	crdCreationParams := metapipeline.CRDCreationParameters{
		Namespace:      ns,
		Context:        o.Context,
		PipelineName:   pipelineName,
		PipelineKind:   o.PipelineKind,
		BuildNumber:    buildNumber,
		GitInfo:        gitInfo,
		Branch:         o.Branch,
		PullRef:        o.PullRefs,
		Revision:       o.Revision,
		SourceDir:      o.SourceName,
		PodTemplates:   podTemplates,
		Trigger:        o.Trigger,
		ServiceAccount: o.ServiceAccount,
		Labels:         labels,
		EnvVars:        envVars,
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
	if o.GitCloneURL == "" {
		return util.MissingOption(gitCloneURLOptionName)
	}

	_, err := gits.ParseGitURL(o.GitCloneURL)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to determine needed git info from the specified git url '%s'", o.GitCloneURL))
	}

	if o.Branch == "" {
		return util.MissingOption(branchOptionName)
	}

	if o.PipelineKind == "" {
		return util.MissingOption(pipelineKindOptionName)
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
	pipelineActivity := tekton.GeneratePipelineActivity(buildNumber, branch, gitInfo, pr)
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
			log.Infof("applied tekton CRDs for %s\n", tektonCRDs.PipelineRun().Name)
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

func (o *StepCreatePipelineOptions) buildLabels(gitInfo *gits.GitRepository) map[string]string {
	// start with custom labels
	labels, _ := util.ExtractKeyValuePairs(o.CustomLabels, "=")

	// add some labels we always want to have based on the build information available
	labels["owner"] = gitInfo.Organisation
	labels["repo"] = gitInfo.Name
	labels["branch"] = o.Branch
	if o.Context != "" {
		labels["context"] = o.Context
	}

	return labels
}

func (o *StepCreatePipelineOptions) buildEnvVars() []corev1.EnvVar {
	var envVars []corev1.EnvVar

	vars, _ := util.ExtractKeyValuePairs(o.CustomEnvs, "=")
	for key, value := range vars {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	return envVars
}

func (o *StepCreatePipelineOptions) buildPullRevs(pullRefs string) (*prow.PullRefs, error) {
	var pr *prow.PullRefs
	var err error
	if pullRefs != "" {
		pr, err = prow.ParsePullRefs(pullRefs)
	}
	return pr, err
}
