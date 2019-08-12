package metapipeline

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/jenkins-x/jx/pkg/tekton"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kubeclient "k8s.io/client-go/kubernetes"
	"os"
	"time"
)

const (
	defaultContainerImage = "jx"
	defaultServiceAccount = "tekton-bot"
	retryDuration         = time.Second * 30
	defaultCheckoutDir    = "source"
)

var (
	logger = log.Logger().WithFields(logrus.Fields{"component": "meta-pipeline-client"})
)

// ClientFactory implements the interface methods to create and apply the meta pipeline.
type ClientFactory struct {
	jxClient         versioned.Interface
	tektonClient     tektonclient.Interface
	kubeClient       kubernetes.Interface
	versionDir       string
	ns               string
	defaultImage     string
	serviceAccount   string
	versionStreamURL string
	versionStreamRef string
}

// NewMetaPipelineClient creates a new client for the creation and application of meta pipelines.
// The responsibility of the meta pipeline is to prepare the execution pipeline and to allow Apps to contribute
// the this execution pipeline.
func NewMetaPipelineClient() (*ClientFactory, error) {
	tektonClient, jxClient, kubeClient, ns, err := getClientsAndNamespace()
	if err != nil {
		return nil, err
	}

	return NewMetaPipelineClientWithClientsAndNamespace(jxClient, tektonClient, kubeClient, ns)
}

// NewMetaPipelineClientWithClientsAndNamespace creates a new client for the creation and application of meta pipelines using the specified parameters.
func NewMetaPipelineClientWithClientsAndNamespace(jxClient versioned.Interface, tektonClient tektonclient.Interface, kubeClient kubernetes.Interface, ns string) (*ClientFactory, error) {
	client := ClientFactory{
		jxClient:       jxClient,
		tektonClient:   tektonClient,
		kubeClient:     kubeClient,
		ns:             ns,
		defaultImage:   defaultContainerImage,
		serviceAccount: defaultServiceAccount,
	}

	return &client, nil
}

// Create creates the Tekton CRDs needed for executing the pipeline as defined by the input parameters
func (c *ClientFactory) Create(pullRef PullRef, pipelineKind PipelineKind, context string, envs map[string]string, labels map[string]string) (kube.PromoteStepActivityKey, tekton.CRDWrapper, error) {
	err := c.cloneVersionStreamIfNeeded()
	if err != nil {
		return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, errors.Wrap(err, "unable to clone version stream")
	}

	gitInfo, err := gits.ParseGitURL(pullRef.SourceURL())
	if err != nil {
		return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, errors.Wrap(err, fmt.Sprintf("unable to determine needed git info from the specified git url '%s'", pullRef.SourceURL()))
	}

	podTemplates, err := c.getPodTemplates(apps.AppPodTemplateName)
	if err != nil {
		return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, errors.Wrap(err, "unable to retrieve pod templates")
	}

	branchIdentifier, err := c.determineBranchIdentifier(pipelineKind, pullRef)
	if err != nil {
		return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, errors.Wrap(err, "unable to create branch identifier")
	}

	// resourceName is shared across all builds of a branch, while the pipelineName is unique for each build.
	resourceName := tekton.PipelineResourceNameFromGitInfo(gitInfo, branchIdentifier, context, tekton.MetaPipeline, nil, "")
	pipelineName := tekton.PipelineResourceNameFromGitInfo(gitInfo, branchIdentifier, context, tekton.MetaPipeline, c.tektonClient, c.ns)
	buildNumber, err := tekton.GenerateNextBuildNumber(c.tektonClient, c.jxClient, c.ns, gitInfo, branchIdentifier, retryDuration, context)
	if err != nil {
		return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, errors.Wrap(err, "unable to determine next build number")
	}

	logger.WithField("repo", gitInfo.URL).WithField("buildNumber", buildNumber).Info("creating meta pipeline CRDs ")

	extendingApps, err := GetExtendingApps(c.jxClient, c.ns)
	if err != nil {
		return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, err
	}

	crdCreationParams := CRDCreationParameters{
		Namespace:        c.ns,
		Context:          context,
		PipelineName:     pipelineName,
		ResourceName:     resourceName,
		PipelineKind:     pipelineKind,
		BuildNumber:      buildNumber,
		BranchIdentifier: branchIdentifier,
		PullRef:          pullRef,
		SourceDir:        defaultCheckoutDir,
		PodTemplates:     podTemplates,
		ServiceAccount:   c.serviceAccount,
		Labels:           labels,
		EnvVars:          envs,
		DefaultImage:     c.defaultImage,
		Apps:             extendingApps,
		VersionsDir:      c.versionDir,
		GitInfo:          *gitInfo,
	}
	tektonCRDs, err := CreateMetaPipelineCRDs(crdCreationParams)
	if err != nil {
		return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, errors.Wrap(err, "failed to generate Tekton CRDs for meta pipeline")
	}

	pipelineActivity := tekton.GeneratePipelineActivity(buildNumber, branchIdentifier, gitInfo, &prow.PullRefs{})

	return *pipelineActivity, *tektonCRDs, nil
}

// Apply takes the given CRDs to process them, usually applying them to the cluster.
func (c *ClientFactory) Apply(pipelineActivity kube.PromoteStepActivityKey, crds tekton.CRDWrapper) error {
	err := tekton.ApplyPipeline(c.jxClient, c.tektonClient, &crds, c.ns, &pipelineActivity)
	if err != nil {
		return errors.Wrapf(err, "failed to apply Tekton CRDs")
	}
	logger.WithField("pipeline", crds.PipelineRun().Name).Info("applied tekton CRDs")
	return nil
}

// Close cleans up the resources use by this client.
func (c *ClientFactory) Close() error {
	return os.RemoveAll(c.versionDir)
}

//DefaultImage returns the default image used for pipeline tasks if no other image is specified.
func (c *ClientFactory) DefaultImage() string {
	return c.defaultImage
}

// SetDefaultImage sets a default image.
func (c *ClientFactory) SetDefaultImage(image string) {
	c.defaultImage = image
}

// ServiceAccount returns the service account under which to execute the pipeline.
func (c *ClientFactory) ServiceAccount() string {
	return c.serviceAccount
}

// SetServiceAccount sets a non default service account for running the Tekton pipeline.
func (c *ClientFactory) SetServiceAccount(serviceAccount string) {
	c.serviceAccount = serviceAccount
}

func (c *ClientFactory) getPodTemplates(containerName string) (map[string]*corev1.Pod, error) {
	podTemplates, err := kube.LoadPodTemplates(c.kubeClient, c.ns)
	if err != nil {
		return nil, err
	}

	return podTemplates, nil
}

func (c *ClientFactory) determineBranchIdentifier(pipelineType PipelineKind, pullRef PullRef) (string, error) {
	var branch string
	switch pipelineType {
	case ReleasePipeline:
		{
			// no pull requests to merge, taking base branch name as identifier
			branch = pullRef.baseBranch
		}
	case PullRequestPipeline:
		{
			if len(pullRef.pullRequests) == 0 {
				return "", errors.New("pullrequest pipeline requested, but no pull requests specified")
			}
			branch = fmt.Sprintf("PR-%s", pullRef.PullRequests()[0].ID)
		}
	default:
		{
			branch = "unknown"
		}
	}
	return branch, nil
}

func (c *ClientFactory) versionStreamURLAndRef() (string, string, error) {
	devEnv, err := kube.GetDevEnvironment(c.jxClient, c.ns)
	if err != nil {
		return "", "", errors.Wrap(err, "unable to retrieve team environment")
	}

	if devEnv == nil {
		return config.DefaultVersionsURL, config.DefaultVersionsRef, nil
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

	return url, ref, nil
}

func (c *ClientFactory) cloneVersionStreamIfNeeded() error {
	url, ref, err := c.versionStreamURLAndRef()
	if err != nil {
		return err
	}

	if c.versionStreamURL != url || c.versionStreamRef != ref {
		_ = os.RemoveAll(c.versionDir)
		c.versionDir, err = c.cloneVersionStream(url, ref)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ClientFactory) cloneVersionStream(url string, ref string) (string, error) {
	dir, err := ioutil.TempDir("", "jx-version-repo-")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temp dir for version stream")
	}

	logger.Infof("cloning version stream url: %s ref: %s into %s", url, ref, dir)

	// Not using GitCLi Clone/ShallowClone atm, since it does not work with tags (HF)
	args := []string{"clone", "--depth", "1", "--branch", ref, url, "."}
	cmd := util.Command{
		Dir:  dir,
		Name: "git",
		Args: args,
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return "", errors.Wrapf(err, "unable to clone version stream: %s", output)
	}

	return dir, err
}

func getClientsAndNamespace() (tektonclient.Interface, jxclient.Interface, kubeclient.Interface, string, error) {
	factory := jxfactory.NewFactory()

	tektonClient, _, err := factory.CreateTektonClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create Tekton client")
	}

	jxClient, _, err := factory.CreateJXClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create JX client")
	}

	kubeClient, ns, err := factory.CreateKubeClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create Kube client")
	}
	ns, _, err = kube.GetDevNamespace(kubeClient, ns)
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to find the dev namespace")
	}
	return tektonClient, jxClient, kubeClient, ns, nil
}
