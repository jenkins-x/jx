package controller

import (
	"encoding/json"
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/cmd/step/git"

	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/spf13/cobra"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/pod-utils/downwardapi"
)

const (
	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"
)

// PipelineRunnerOptions holds the command line arguments
type PipelineRunnerOptions struct {
	*opts.CommonOptions
	BindAddress          string
	Path                 string
	Port                 int
	NoGitCredentialsInit bool
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
	controllerPipelineRunnersLong = templates.LongDesc(`Runs the service to generate Tekton PipelineRun resources from source code webhooks such as from Prow`)

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
	cmd.Flags().StringVarP(&options.BindAddress, optionBind, "", "",
		"The interface address to bind to (by default, will listen on all interfaces/addresses).")
	cmd.Flags().StringVarP(&options.Path, "path", "p", "/",
		"The path to listen on for requests to trigger a pipeline run.")
	cmd.Flags().StringVarP(&options.ServiceAccount, "service-account", "", "tekton-bot", "The Kubernetes ServiceAccount to use to run the pipeline")
	cmd.Flags().BoolVarP(&options.NoGitCredentialsInit, "no-git-init", "", false, "Disables checking we have setup git credentials on startup")
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
	mux := http.NewServeMux()
	mux.Handle(o.Path, http.HandlerFunc(o.pipelineRunMethods))
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))
	log.Logger().Infof("waiting for dynamic Tekton Pipelines at http://%s:%d%s", o.BindAddress, o.Port, o.Path)
	return http.ListenAndServe(":"+strconv.Itoa(o.Port), mux)
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *PipelineRunnerOptions) health(w http.ResponseWriter, r *http.Request) {
	log.Logger().Trace("health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *PipelineRunnerOptions) ready(w http.ResponseWriter, r *http.Request) {
	log.Logger().Trace("ready check")
	w.WriteHeader(http.StatusNoContent)
}

// handle request for pipeline runs
func (o *PipelineRunnerOptions) pipelineRunMethods(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_, err := fmt.Fprintf(w, "please POST JSON to this endpoint!\n")
		if err != nil {
			log.Logger().Errorf("unable to write response to GET request: %s", err.Error())
		}
	case http.MethodHead:
		log.Logger().Info("HEAD Todo...")
	case http.MethodPost:
		o.handlePostRequest(r, w)
	default:
		log.Logger().Errorf("unsupported method %s for %s", r.Method, o.Path)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (o *PipelineRunnerOptions) handlePostRequest(r *http.Request, w http.ResponseWriter) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Logger().Warn("Unable to dump request for debugging purposes")
	}
	log.Logger().Info(string(requestDump))
	requestParams, err := o.parsePipelineRequestParams(r)
	if err != nil {
		o.returnError(err, "could not read the JSON request body: "+err.Error(), w)
		return
	}

	pipelineRunResponse, err := o.startPipeline(requestParams)
	if err != nil {
		o.returnError(err, "could not start pipeline: "+err.Error(), w)
		return
	}

	data, err := o.marshalPayload(pipelineRunResponse)
	if err != nil {
		o.returnError(err, "failed to marshal payload", w)
		return
	}
	_, err = w.Write(data)
	if err != nil {
		log.Logger().Errorf("error writing PipelineRunResponse: %s", err.Error())
	}
}

func (o *PipelineRunnerOptions) parsePipelineRequestParams(r *http.Request) (PipelineRunRequest, error) {
	request := PipelineRunRequest{}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return request, errors.Wrapf(err, fmt.Sprintf("could not read the JSON request body: %s", err.Error()))
	}
	err = json.Unmarshal(data, &request)
	if err != nil {
		return request, errors.Wrapf(err, fmt.Sprintf("failed to unmarshal the JSON request body: %s", err.Error()))
	}
	log.Logger().Debugf("got payload %s", util.PrettyPrint(request))
	return request, nil
}

// startPipeline handles an incoming request to start a pipeline.
func (o *PipelineRunnerOptions) startPipeline(requestParams PipelineRunRequest) (PipelineRunResponse, error) {
	err := o.stepGitCredentials()
	if err != nil {
		log.Logger().Warn(err.Error())
	}

	response := PipelineRunResponse{}
	var revision string
	var prNumber string

	prowJobSpec := requestParams.ProwJobSpec
	if prowJobSpec.Refs == nil {
		return response, errors.New(fmt.Sprintf("no prowJobSpec.refs passed in so cannot determine git repository: %s", util.PrettyPrint(requestParams)))
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

	sourceURL := fmt.Sprintf("https://github.com/%s/%s.git", prowJobSpec.Refs.Org, prowJobSpec.Refs.Repo)
	if sourceURL == "" {
		return response, errors.Wrap(err, "missing sourceURL property")
	}

	if revision == "" {
		revision = "master"
	}

	pr := &create.StepCreateTaskOptions{}
	if prowJobSpec.Type == prowapi.PostsubmitJob {
		pr.PipelineKind = jenkinsfile.PipelineKindRelease
	} else {
		pr.PipelineKind = jenkinsfile.PipelineKindPullRequest
	}

	branch := getBranch(prowJobSpec)
	if branch == "" {
		branch = "master"
	}

	copy := *o.CommonOptions
	pr.CommonOptions = &copy

	// defaults
	pr.SourceName = "source"
	pr.Duration = time.Second * 20
	pr.Trigger = string(pipelineapi.PipelineTriggerTypeManual)
	pr.PullRequestNumber = prNumber
	pr.CloneGitURL = sourceURL
	pr.DeleteTempDir = true
	pr.Context = prowJobSpec.Context
	pr.Branch = branch
	pr.Revision = revision
	pr.ServiceAccount = o.ServiceAccount

	// turn map into string array with = separator to match type of custom labels which are CLI flags
	for key, value := range requestParams.Labels {
		pr.CustomLabels = append(pr.CustomLabels, fmt.Sprintf("%s=%s", key, value))
	}

	// turn map into string array with = separator to match type of custom env vars which are CLI flags
	for key, value := range envs {
		pr.CustomEnvs = append(pr.CustomEnvs, fmt.Sprintf("%s=%s", key, value))
	}

	log.Logger().Infof("triggering pipeline for repo %s branch %s revision %s context %s", sourceURL, branch, revision, prowJobSpec.Context)

	err = pr.Run()
	if err != nil {
		return response, errors.Wrap(err, "error triggering the pipeline run")
	}

	results := PipelineRunResponse{
		Resources: pr.Results.ObjectReferences(),
	}
	return results, nil
}

func (o *PipelineRunnerOptions) marshalPayload(payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "marshalling the JSON payload %#v", payload)
	}
	return data, nil
}

func (o *PipelineRunnerOptions) returnError(err error, message string, w http.ResponseWriter) {
	log.Logger().Errorf("%v %s", err, message)
	w.WriteHeader(400)
	w.Write([]byte(message))
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

func getBranch(spec prowapi.ProwJobSpec) string {
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
