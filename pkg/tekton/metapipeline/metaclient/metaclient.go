package metaclient

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/jenkins-x/jx/pkg/apps"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	kubeclient "k8s.io/client-go/kubernetes"
)

const (
	jobOptionName       = "job"
	pullRefOptionName   = "pull-refs"
	sourceURLOptionName = "source-url"

	retryDuration      = time.Second * 30
	defaultCheckoutDir = "source"
)

// MetaClient contains the arguments to create the meta pipeline
type MetaClient struct {
	SourceURL      string
	PullRefs       string
	Context        string
	Job            string
	ServiceAccount string
	CustomLabels   []string
	CustomEnvs     []string
	DefaultImage   string

	Results     tekton.CRDWrapper
	OutDir      string
	Verbose     bool
	NoApply     *bool
	versionsDir string
	git         gits.Gitter
	factory     jxfactory.Factory
}

// Run creates the meta pipeline
func (o *MetaClient) Run() error {
	err := o.validateArguments()
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

	// resourceName is shared across all builds of a branch, while the pipelineName is unique for each build.
	resourceName := tekton.PipelineResourceNameFromGitInfo(gitInfo, branchIdentifier, o.Context, tekton.MetaPipeline, nil, "")
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

	if o.versionsDir == "" {
		o.versionsDir, err = o.gitCloneVersionStream(jxClient, ns)
		if err != nil {
			return err
		}

		defer os.RemoveAll(o.versionsDir)
	}

	crdCreationParams := metapipeline.CRDCreationParameters{
		Namespace:        ns,
		Context:          o.Context,
		PipelineName:     pipelineName,
		ResourceName:     resourceName,
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
		VersionsDir:      o.versionsDir,
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

func (o *MetaClient) validateArguments() error {
	// default values
	if o.DefaultImage == "" {
		o.DefaultImage = syntax.DefaultContainerImage
	}
	if o.ServiceAccount == "" {
		o.ServiceAccount = "tekton-bot"
	}
	if o.OutDir == "" {
		o.OutDir = "out"
	}

	if o.SourceURL == "" {
		return util.MissingOption(sourceURLOptionName)
	}
	if o.Job == "" {
		return util.MissingOption(jobOptionName)
	}
	if o.PullRefs == "" {
		return util.MissingOption(pullRefOptionName)
	}

	_, err := util.ExtractKeyValuePairs(o.CustomLabels, "=")
	if err != nil {
		return errors.Wrap(err, "unable to parse custom labels")
	}
	_, err = util.ExtractKeyValuePairs(o.CustomEnvs, "=")
	if err != nil {
		return errors.Wrap(err, "unable to parse custom environment variables")
	}
	return nil
}

func (o *MetaClient) handleResult(tektonClient tektonclient.Interface,
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

func (o *MetaClient) getPodTemplates(kubeClient kubeclient.Interface, ns string, containerName string) (map[string]*corev1.Pod, error) {
	podTemplates, err := kube.LoadPodTemplates(kubeClient, ns)
	if err != nil {
		return nil, err
	}
	return podTemplates, nil
}

// GetFactory returns the factory
func (o *MetaClient) GetFactory() jxfactory.Factory {
	if o.factory == nil {
		o.factory = jxfactory.NewFactory()
	}
	return o.factory
}

func (o *MetaClient) getClientsAndNamespace() (tektonclient.Interface, jxclient.Interface, kubeclient.Interface, string, error) {
	f := o.GetFactory()

	tektonClient, _, err := f.CreateTektonClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create Tekton client")
	}

	jxClient, _, err := f.CreateJXClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create JX client")
	}

	kubeClient, ns, err := f.CreateKubeClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create Kube client")
	}
	ns, _, err = kube.GetDevNamespace(kubeClient, ns)
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to find the dev namespace")
	}
	return tektonClient, jxClient, kubeClient, ns, nil
}

// determineBranchIdentifier finds a identifier for the branch which is build by the specified pull ref.
// The name is either an actual git branch name (eg master) or a synthetic names  like PR-<number> for single pull requests
// or 'batch' for Prow batch builds.
// The method makes the decision purely based on the Prow pull ref. At this stage we don't have the full ProwJov spec
// available.
func (o *MetaClient) determineBranchIdentifier(pullRef prow.PullRefs) string {
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

func (o *MetaClient) determinePipelineKind(pullRef prow.PullRefs) string {
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

func (o *MetaClient) gitCloneVersionStream(jxClient jxclient.Interface, ns string) (string, error) {
	dir, err := ioutil.TempDir("", "jx-version-repo-")
	if err != nil {
		return dir, err
	}
	os.RemoveAll(dir)

	devEnv, err := kube.GetDevEnvironment(jxClient, ns)
	if err != nil {
		return dir, err
	}
	teamSettings := devEnv.Spec.TeamSettings
	url := teamSettings.VersionStreamURL
	ref := teamSettings.VersionStreamRef
	if url == "" {
		url = config.DefaultVersionsURL
	}
	if ref == "" {
		ref = config.DefaultVersionsRef
	}
	log.Logger().WithField("url", url).WithField("ref", ref).Infof("shallow cloning version stream")
	err = o.Git().ShallowClone(dir, url, ref, "")
	return dir, err
}

// Git returns the git client
func (o *MetaClient) Git() gits.Gitter {
	if o.git == nil {
		o.git = gits.NewGitCLI()
	}
	return o.git
}

// SetGit sets the git client
func (o *MetaClient) SetGit(git gits.Gitter) {
	o.git = git
}
