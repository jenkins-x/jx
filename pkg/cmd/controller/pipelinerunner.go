package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/cmd/step/git"

	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/log"

	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/spf13/cobra"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/pod-utils/downwardapi"
)

const (
	// healthPath is the URL path for the HTTP endpoint that returns health status.
	healthPath = "/health"
	// readyPath URL path for the HTTP endpoint that returns ready status.
	readyPath = "/ready"

	// jobLabel is the label name used to identify the Prow job within PipelineRunRequest.Labels
	jobLabel = "prowJobName"

	shutdownTimeout = 5
)

var logger = log.Logger().WithFields(logrus.Fields{"component": "pipelinerunner"})

// PipelineRunnerOptions holds the command line arguments
type PipelineRunnerOptions struct {
	*opts.CommonOptions
	BindAddress          string
	Path                 string
	Port                 int
	NoGitCredentialsInit bool
	UseMetaPipeline      bool
	MetaPipelineImage    string
	SemanticRelease      bool
}

// PipelineRunRequest the request to trigger a pipeline run
type PipelineRunRequest struct {
	Labels      map[string]string   `json:"labels,omitempty"`
	ProwJobSpec prowapi.ProwJobSpec `json:"prowJobSpec,omitempty"`
}

// PipelineRunResponse the results of triggering a pipeline run
type PipelineRunResponse struct {
	Resources []kube.ObjectReference `json:"resources,omitempty"`
}

// ObjectReference represents a reference to a k8s resource
type ObjectReference struct {
	APIVersion string `json:"apiVersion" protobuf:"bytes,5,opt,name=apiVersion"`
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	Kind string `json:"kind" protobuf:"bytes,1,opt,name=kind"`
	// Name of the referent.
	// More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`
}

var (
	controllerPipelineRunnersLong = templates.LongDesc(`Runs the service to generate Tekton resources from source code webhooks such as from Prow`)

	controllerPipelineRunnersExample = templates.Examples(`
			# run the pipeline runner controller
			jx controller pipelinerunner
		`)
)

// NewCmdControllerPipelineRunner creates the command
func NewCmdControllerPipelineRunner(commonOpts *opts.CommonOptions) *cobra.Command {
	options := PipelineRunnerOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "pipelinerunner",
		Short:   "Runs the service to generate Tekton PipelineRun resources from source code webhooks such as from Prow",
		Long:    controllerPipelineRunnersLong,
		Example: controllerPipelineRunnersExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().IntVarP(&options.Port, optionPort, "", 8080, "The TCP port to listen on.")
	cmd.Flags().StringVarP(&options.BindAddress, optionBind, "", "0.0.0.0", "The interface address to bind to (by default, will listen on all interfaces/addresses).")
	cmd.Flags().StringVarP(&options.Path, "path", "p", "/", "The path to listen on for requests to trigger a pipeline run.")
	cmd.Flags().StringVarP(&options.ServiceAccount, "service-account", "", "tekton-bot", "The Kubernetes ServiceAccount to use to run the pipeline.")
	cmd.Flags().BoolVarP(&options.NoGitCredentialsInit, "no-git-init", "", false, "Disables checking we have setup git credentials on startup.")
	cmd.Flags().BoolVarP(&options.SemanticRelease, "semantic-release", "", false, "Enable semantic releases")

	// TODO - temporary flags until meta pipeline is the default
	cmd.Flags().BoolVarP(&options.UseMetaPipeline, "use-meta-pipeline", "", false, "Uses the meta pipeline to create the pipeline.")
	cmd.Flags().StringVar(&options.MetaPipelineImage, "meta-pipeline-image", "", "Specify the docker image to use if there is no image specified for a step.")
	return cmd
}

// Run will implement this command
func (o *PipelineRunnerOptions) Run() error {
	if !o.NoGitCredentialsInit {
		err := o.InitGitConfigAndUser()
		if err != nil {
			return err
		}
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.startWorkers(ctx, &wg, cancel)
	o.setupSignalChannel(cancel)
	wg.Wait()
	return nil
}

func (o *PipelineRunnerOptions) startWorkers(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		mux := http.NewServeMux()
		mux.Handle(o.Path, http.HandlerFunc(o.pipeline))
		mux.Handle(healthPath, http.HandlerFunc(o.health))
		mux.Handle(readyPath, http.HandlerFunc(o.ready))
		srv := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", o.BindAddress, o.Port),
			Handler: mux,
		}

		go func() {
			logger.Infof("Starting HTTP Server on %s port %d", o.BindAddress, o.Port)
			if err := srv.ListenAndServe(); err != nil {
				cancel()
				return
			}
		}()

		for {
			select {
			case <-ctx.Done():
				logger.Info("Shutting down HTTP server")
				ctx, cancel := context.WithTimeout(ctx, shutdownTimeout*time.Second)
				_ = srv.Shutdown(ctx)
				cancel()
				return
			}
		}
	}()
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *PipelineRunnerOptions) health(w http.ResponseWriter, r *http.Request) {
	logger.Trace("health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *PipelineRunnerOptions) ready(w http.ResponseWriter, r *http.Request) {
	logger.Trace("ready check")
	w.WriteHeader(http.StatusNoContent)
}

// handle request for pipeline runs
func (o *PipelineRunnerOptions) pipeline(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_, err := fmt.Fprintf(w, "please POST JSON to this endpoint!\n")
		if err != nil {
			logger.Errorf("unable to write response to GET request: %s", err.Error())
		}
	case http.MethodHead:
		logger.Info("HEAD Todo...")
	case http.MethodPost:
		requestDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			logger.Warn("Unable to log POST request")
		}
		logger.WithFields(logrus.Fields{"request": string(requestDump)}).Info("POST request")

		o.handlePostRequest(r, w)
	default:
		logger.Errorf("unsupported method %s for %s", r.Method, o.Path)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (o *PipelineRunnerOptions) handlePostRequest(r *http.Request, w http.ResponseWriter) {
	requestParams, err := o.parseStartPipelineRequestParameters(r)
	if err != nil {
		o.returnStatusBadRequest(err, "could not read the JSON request body: "+err.Error(), w)
		return
	}

	pipelineRunResponse, err := o.startPipeline(requestParams)
	if err != nil {
		o.returnStatusBadRequest(err, "could not start pipeline: "+err.Error(), w)
		return
	}

	data, err := o.marshalPayload(pipelineRunResponse)
	if err != nil {
		o.returnStatusBadRequest(err, "failed to marshal payload", w)
		return
	}
	_, err = w.Write(data)
	if err != nil {
		logger.Errorf("error writing PipelineRunResponse: %s", err.Error())
	}
}

func (o *PipelineRunnerOptions) parseStartPipelineRequestParameters(r *http.Request) (PipelineRunRequest, error) {
	request := PipelineRunRequest{}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return request, errors.Wrapf(err, fmt.Sprintf("could not read the JSON request body: %s", err.Error()))
	}
	err = json.Unmarshal(data, &request)
	if err != nil {
		return request, errors.Wrapf(err, fmt.Sprintf("failed to unmarshal the JSON request body: %s", err.Error()))
	}
	logger.Debugf("got payload %s", util.PrettyPrint(request))
	return request, nil
}

// startPipeline handles an incoming request to start a pipeline.
func (o *PipelineRunnerOptions) startPipeline(pipelineRun PipelineRunRequest) (PipelineRunResponse, error) {
	err := o.stepGitCredentials()
	if err != nil {
		logger.Warn(err.Error())
	}

	response := PipelineRunResponse{}
	var revision string
	var prNumber string

	prowJobSpec := pipelineRun.ProwJobSpec
	if prowJobSpec.Refs == nil {
		return response, errors.New(fmt.Sprintf("no prowJobSpec.refs passed: %s", util.PrettyPrint(pipelineRun)))
	}

	// Only if there is one Pull in Refs, it's a PR build so we are going to pass it
	if len(prowJobSpec.Refs.Pulls) == 1 {
		revision = prowJobSpec.Refs.Pulls[0].SHA
		prNumber = strconv.Itoa(prowJobSpec.Refs.Pulls[0].Number)
	} else {
		//Otherwise it's a Master / Batch build, and we handle it later
		revision = prowJobSpec.Refs.BaseSHA
	}

	envs, err := downwardapi.EnvForSpec(downwardapi.NewJobSpec(prowJobSpec, "", ""))
	if err != nil {
		return response, errors.Wrap(err, "failed to get env vars from prowjob")
	}

	sourceURL := o.getSourceURL(prowJobSpec.Refs.Org, prowJobSpec.Refs.Repo)
	if sourceURL == "" {
		sourceURL = fmt.Sprintf("https://github.com/%s/%s.git", prowJobSpec.Refs.Org, prowJobSpec.Refs.Repo)
	}
	if sourceURL == "" {
		return response, errors.Wrap(err, "missing sourceURL property")
	}

	if revision == "" {
		revision = "master"
	}

	branch := o.getBranch(prowJobSpec)
	if branch == "" {
		branch = "master"
	}

	logger.WithFields(logrus.Fields{"sourceURL": sourceURL, "branch": branch, "revision": revision, "context": prowJobSpec.Context, "meta": o.UseMetaPipeline}).Info("triggering pipeline")

	results := PipelineRunResponse{}
	if o.UseMetaPipeline {
		pipelineCreateOption, err := o.buildStepCreatePipelineOption(pipelineRun, prNumber, sourceURL, revision, branch, envs)
		if err != nil {
			return response, errors.Wrap(err, "error creating options for creating meta pipeline")
		}

		err = pipelineCreateOption.Run()
		if err != nil {
			return response, errors.Wrap(err, "error triggering the pipeline run")
		}
		results.Resources = pipelineCreateOption.Results.ObjectReferences()
	} else {
		pipelineCreateOption := o.buildStepCreateTaskOption(prowJobSpec, prNumber, sourceURL, revision, branch, pipelineRun, envs)
		err = pipelineCreateOption.Run()
		if err != nil {
			return response, errors.Wrap(err, "error triggering the pipeline run")
		}
		results.Resources = pipelineCreateOption.Results.ObjectReferences()
	}

	return results, nil
}

func (o *PipelineRunnerOptions) buildStepCreateTaskOption(prowJobSpec prowapi.ProwJobSpec, prNumber string, sourceURL string, revision string, branch string, pipelineRun PipelineRunRequest, envs map[string]string) *create.StepCreateTaskOptions {
	createTaskOption := &create.StepCreateTaskOptions{}
	if prowJobSpec.Type == prowapi.PostsubmitJob {
		createTaskOption.PipelineKind = jenkinsfile.PipelineKindRelease
	} else {
		createTaskOption.PipelineKind = jenkinsfile.PipelineKindPullRequest
	}

	c := *o.CommonOptions
	createTaskOption.CommonOptions = &c
	// defaults
	createTaskOption.SourceName = "source"
	createTaskOption.Duration = time.Second * 20
	createTaskOption.PullRequestNumber = prNumber
	createTaskOption.CloneGitURL = sourceURL
	createTaskOption.DeleteTempDir = true
	createTaskOption.Context = prowJobSpec.Context
	createTaskOption.Branch = branch
	createTaskOption.Revision = revision
	createTaskOption.ServiceAccount = o.ServiceAccount
	createTaskOption.SemanticRelease = o.SemanticRelease
	// turn map into string array with = separator to match type of custom labels which are CLI flags
	for key, value := range pipelineRun.Labels {
		createTaskOption.CustomLabels = append(createTaskOption.CustomLabels, fmt.Sprintf("%s=%s", key, value))
	}
	// turn map into string array with = separator to match type of custom env vars which are CLI flags
	for key, value := range envs {
		createTaskOption.CustomEnvs = append(createTaskOption.CustomEnvs, fmt.Sprintf("%s=%s", key, value))
	}

	return createTaskOption
}

func (o *PipelineRunnerOptions) buildStepCreatePipelineOption(pipelineRun PipelineRunRequest, prNumber string, sourceURL string, revision string, branch string, envs map[string]string) (*create.StepCreatePipelineOptions, error) {
	prowJobSpec := pipelineRun.ProwJobSpec
	pullRefs := o.getPullRefs(prowJobSpec)

	job := pipelineRun.Labels[jobLabel]
	if job == "" {
		return nil, errors.Errorf("unable to find prow job name in pipeline request: %s", util.PrettyPrint(pipelineRun))
	}

	createPipelineOption := &create.StepCreatePipelineOptions{}
	c := *o.CommonOptions
	createPipelineOption.CommonOptions = &c
	createPipelineOption.SourceURL = sourceURL
	createPipelineOption.PullRefs = pullRefs.String()
	createPipelineOption.Context = prowJobSpec.Context
	createPipelineOption.Job = job

	createPipelineOption.ServiceAccount = o.ServiceAccount
	createPipelineOption.DefaultImage = o.MetaPipelineImage

	// turn map into string array with = separator to match type of custom labels which are CLI flags
	for key, value := range pipelineRun.Labels {
		createPipelineOption.CustomLabels = append(createPipelineOption.CustomLabels, fmt.Sprintf("%s=%s", key, value))
	}
	// turn map into string array with = separator to match type of custom env vars which are CLI flags
	for key, value := range envs {
		createPipelineOption.CustomEnvs = append(createPipelineOption.CustomEnvs, fmt.Sprintf("%s=%s", key, value))
	}

	return createPipelineOption, nil
}

func (o *PipelineRunnerOptions) marshalPayload(payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "marshalling the JSON payload %#v", payload)
	}
	return data, nil
}

func (o *PipelineRunnerOptions) returnStatusBadRequest(err error, message string, w http.ResponseWriter) {
	logger.Infof("%v %s", err, message)
	w.WriteHeader(http.StatusBadRequest)
	_, err = w.Write([]byte(message))
	if err != nil {
		logger.Warnf("Error returning HTTP 400: %s", err)
	}
}

func (o *PipelineRunnerOptions) stepGitCredentials() error {
	if !o.NoGitCredentialsInit {
		copy := *o.CommonOptions
		copy.BatchMode = true
		gsc := &git.StepGitCredentialsOptions{
			StepOptions: opts.StepOptions{
				CommonOptions: &copy,
			},
		}
		err := gsc.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to run: jx step git credentials")
		}
	}
	return nil
}

func (o *PipelineRunnerOptions) getBranch(spec prowapi.ProwJobSpec) string {
	branch := spec.Refs.BaseRef
	if spec.Type == prowapi.PostsubmitJob {
		return branch
	}
	if spec.Type == prowapi.BatchJob {
		return "batch"
	}
	if len(spec.Refs.Pulls) > 0 {
		branch = fmt.Sprintf("PR-%v", spec.Refs.Pulls[0].Number)
	}
	return branch
}

func (o *PipelineRunnerOptions) getPullRefs(spec prowapi.ProwJobSpec) prow.PullRefs {
	toMerge := make(map[string]string)
	for _, pull := range spec.Refs.Pulls {
		toMerge[strconv.Itoa(pull.Number)] = pull.SHA
	}

	pullRef := prow.PullRefs{
		BaseBranch: spec.Refs.BaseRef,
		BaseSha:    spec.Refs.BaseSHA,
		ToMerge:    toMerge,
	}
	return pullRef
}

// setupSignalChannel registers a listener for Unix signals for a ordered shutdown
func (o *PipelineRunnerOptions) setupSignalChannel(cancel context.CancelFunc) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM)

	go func() {
		<-sigchan
		logger.Info("Received SIGTERM signal. Initiating shutdown.")
		cancel()
	}()
}

func (o *PipelineRunnerOptions) getSourceURL(org, repo string) string {
	jxClient, ns, err := o.getClientsAndNamespace()

	if err != nil {
		logger.Debugf("failed to get jxClient or namespace %v", err)
		return ""
	}

	resourceInterface := jxClient.JenkinsV1().SourceRepositories(ns)

	sourceRepos, err := resourceInterface.List(meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("owner=%s,repository=%s", org, repo),
	})

	if err != nil || sourceRepos == nil || len(sourceRepos.Items) == 0 {
		return ""
	}

	gitProviderURL := sourceRepos.Items[0].Spec.Provider
	if gitProviderURL == "" {
		return ""
	}
	if !strings.HasSuffix(gitProviderURL, "/") {
		gitProviderURL = gitProviderURL + "/"
	}

	return fmt.Sprintf("%s%s/%s.git", gitProviderURL, org, repo)
}

func (o *PipelineRunnerOptions) getClientsAndNamespace() (jxclient.Interface, string, error) {

	jxClient, _, err := o.JXClient()
	if err != nil {
		return nil, "", errors.Wrap(err, "unable to create JX client")
	}

	_, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, "", errors.Wrap(err, "unable to create Kube client")
	}

	return jxClient, ns, nil
}
