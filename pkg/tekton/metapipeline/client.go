package metapipeline

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
	"github.com/jenkins-x/jx/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	kubeclient "k8s.io/client-go/kubernetes"
)

const (
	pullRefOptionName   = "pull-refs"
	sourceURLOptionName = "source-url"

	retryDuration      = time.Second * 30
	defaultCheckoutDir = "source"
)

// Client contains the arguments to create the meta pipeline
type Client struct {
	SourceURL      string
	PullRefs       string
	Context        string
	ServiceAccount string
	CustomLabels   []string
	CustomEnvs     []string
	DefaultImage   string

	Results     tekton.CRDWrapper
	OutDir      string
	NoApply     bool
	versionsDir string
	gitter      gits.Gitter
	factory     jxfactory.Factory
}

// Create creates the meta pipeline
func (c *Client) Create() error {
	err := c.validateArguments()
	if err != nil {
		return errors.Wrapf(err, "invalid arguments")
	}

	gitInfo, err := gits.ParseGitURL(c.SourceURL)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to determine needed git info from the specified git url '%s'", c.SourceURL))
	}

	pullRefs, err := prow.ParsePullRefs(c.PullRefs)
	if err != nil {
		return errors.Wrapf(err, "unable to parse pull ref '%s'", c.PullRefs)
	}

	tektonClient, jxClient, kubeClient, ns, err := c.getClientsAndNamespace()
	if err != nil {
		return err
	}

	podTemplates, err := c.getPodTemplates(kubeClient, ns, apps.AppPodTemplateName)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve pod templates")
	}
	c.defaultImageFromPodTemplate(podTemplates)

	branchIdentifier := c.determineBranchIdentifier(*pullRefs)
	pipelineKind := c.determinePipelineKind(*pullRefs)

	// resourceName is shared across all builds of a branch, while the pipelineName is unique for each build.
	resourceName := tekton.PipelineResourceNameFromGitInfo(gitInfo, branchIdentifier, c.Context, tekton.MetaPipeline, nil, "")
	pipelineName := tekton.PipelineResourceNameFromGitInfo(gitInfo, branchIdentifier, c.Context, tekton.MetaPipeline, tektonClient, ns)
	buildNumber, err := tekton.GenerateNextBuildNumber(tektonClient, jxClient, ns, gitInfo, branchIdentifier, retryDuration, c.Context)
	if err != nil {
		return errors.Wrap(err, "unable to determine next build number")
	}

	log.Logger().Debugf("creating meta pipeline CRDs for build %s of repository '%s/%s'", buildNumber, gitInfo.Organisation, gitInfo.Name)

	extendingApps, err := GetExtendingApps(jxClient, ns)
	if err != nil {
		return err
	}

	if c.versionsDir == "" {
		c.versionsDir, err = c.gitCloneVersionStream(jxClient, ns)
		if err != nil {
			return err
		}

		defer os.RemoveAll(c.versionsDir)
	}

	crdCreationParams := CRDCreationParameters{
		Namespace:        ns,
		Context:          c.Context,
		PipelineName:     pipelineName,
		ResourceName:     resourceName,
		PipelineKind:     pipelineKind,
		BuildNumber:      buildNumber,
		GitInfo:          *gitInfo,
		BranchIdentifier: branchIdentifier,
		PullRef:          *pullRefs,
		SourceDir:        defaultCheckoutDir,
		PodTemplates:     podTemplates,
		ServiceAccount:   c.ServiceAccount,
		Labels:           c.CustomLabels,
		EnvVars:          c.CustomEnvs,
		DefaultImage:     c.DefaultImage,
		Apps:             extendingApps,
		VersionsDir:      c.versionsDir,
	}
	tektonCRDs, err := CreateMetaPipelineCRDs(crdCreationParams)
	if err != nil {
		return errors.Wrap(err, "failed to generate Tekton CRDs for meta pipeline")
	}
	// record the results in the struct for the case this command is called programmatically (HF)
	c.Results = *tektonCRDs

	err = c.handleResult(tektonClient, jxClient, tektonCRDs, buildNumber, branchIdentifier, *pullRefs, ns, gitInfo)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) validateArguments() error {
	// default values
	if c.DefaultImage == "" {
		c.DefaultImage = syntax.DefaultContainerImage
	}
	if c.ServiceAccount == "" {
		c.ServiceAccount = "tekton-bot"
	}
	if c.OutDir == "" {
		c.OutDir = "out"
	}

	if c.SourceURL == "" {
		return util.MissingOption(sourceURLOptionName)
	}
	if c.PullRefs == "" {
		return util.MissingOption(pullRefOptionName)
	}

	_, err := util.ExtractKeyValuePairs(c.CustomLabels, "=")
	if err != nil {
		return errors.Wrap(err, "unable to parse custom labels")
	}
	_, err = util.ExtractKeyValuePairs(c.CustomEnvs, "=")
	if err != nil {
		return errors.Wrap(err, "unable to parse custom environment variables")
	}
	return nil
}

func (c *Client) handleResult(tektonClient tektonclient.Interface,
	jxClient jxclient.Interface,
	tektonCRDs *tekton.CRDWrapper,
	buildNumber string,
	branch string,
	pullRefs prow.PullRefs,
	ns string,
	gitInfo *gits.GitRepository) error {

	pipelineActivity := tekton.GeneratePipelineActivity(buildNumber, branch, gitInfo, &pullRefs, tekton.MetaPipeline)
	if c.NoApply {
		err := tektonCRDs.WriteToDisk(c.OutDir, pipelineActivity)
		if err != nil {
			return errors.Wrapf(err, "failed to output Tekton CRDs")
		}
	} else {
		err := tekton.ApplyPipeline(jxClient, tektonClient, tektonCRDs, ns, gitInfo, branch, pipelineActivity)
		if err != nil {
			return errors.Wrapf(err, "failed to apply Tekton CRDs")
		}
		log.Logger().Debugf("applied tekton CRDs for %s", tektonCRDs.PipelineRun().Name)
	}
	return nil
}

func (c *Client) getPodTemplates(kubeClient kubeclient.Interface, ns string, containerName string) (map[string]*corev1.Pod, error) {
	podTemplates, err := kube.LoadPodTemplates(kubeClient, ns)
	if err != nil {
		return nil, err
	}
	return podTemplates, nil
}

// getFactory returns the factory
func (c *Client) getFactory() jxfactory.Factory {
	if c.factory == nil {
		c.factory = jxfactory.NewFactory()
	}
	return c.factory
}

func (c *Client) getClientsAndNamespace() (tektonclient.Interface, jxclient.Interface, kubeclient.Interface, string, error) {
	f := c.getFactory()

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
func (c *Client) determineBranchIdentifier(pullRef prow.PullRefs) string {
	prCount := len(pullRef.ToMerge)
	var branch string
	switch prCount {
	case 0:
		// no pull requests to merge, taking base branch name as identifier
		branch = pullRef.BaseBranch
	case 1:
		// single pull request, create synthetic PR identifier
		for k := range pullRef.ToMerge {
			branch = fmt.Sprintf("PR-%s", k)
			break
		}
	default:
		branch = "batch"
	}
	log.Logger().Debugf("branch identifier for pull ref '%s' : '%s'", pullRef.String(), branch)
	return branch
}

func (c *Client) determinePipelineKind(pullRef prow.PullRefs) string {
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

func (c *Client) gitCloneVersionStream(jxClient jxclient.Interface, ns string) (string, error) {
	dir, err := ioutil.TempDir("", "jx-version-repo-")
	if err != nil {
		return dir, err
	}
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
	err = c.git().ShallowClone(dir, url, ref, "")
	return dir, err
}

// git returns the git client
func (c *Client) git() gits.Gitter {
	if c.gitter == nil {
		c.gitter = gits.NewGitCLI()
	}
	return c.gitter
}

// setGit sets the git client
func (c *Client) setGit(git gits.Gitter) {
	c.gitter = git
}

func (c *Client) defaultImageFromPodTemplate(pods map[string]*corev1.Pod) {
	pod := pods[c.DefaultImage]
	if pod != nil {
		containers := pod.Spec.Containers
		if len(containers) > 0 {
			cont := containers[0]
			image := cont.Image
			if image != "" {
				log.Logger().Debugf("using default image for meta pipeline: %s", image)
				c.DefaultImage = image
			}
		}
	}
}
