package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/jx/cmd/step/git"

	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
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

// ControllerPipelineRunnerOptions holds the command line arguments
type ControllerPipelineRunnerOptions struct {
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
	options := ControllerPipelineRunnerOptions{
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
func (o *ControllerPipelineRunnerOptions) Run() error {
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
	log.Logger().Infof("Waiting for dynamic Tekton Pipelines at http://%s:%d%s", o.BindAddress, o.Port, o.Path)
	return http.ListenAndServe(":"+strconv.Itoa(o.Port), mux)
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *ControllerPipelineRunnerOptions) health(w http.ResponseWriter, r *http.Request) {
	log.Logger().Debug("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *ControllerPipelineRunnerOptions) ready(w http.ResponseWriter, r *http.Request) {
	log.Logger().Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// handle request for pipeline runs
func (o *ControllerPipelineRunnerOptions) pipelineRunMethods(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		fmt.Fprintf(w, "Please POST JSON to this endpoint!\n")
	case http.MethodHead:
		log.Logger().Info("HEAD Todo...")
	case http.MethodPost:
		o.startPipelineRun(w, r)
	default:
		log.Logger().Errorf("Unsupported method %s for %s", r.Method, o.Path)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

// handle request for pipeline runs
func (o *ControllerPipelineRunnerOptions) startPipelineRun(w http.ResponseWriter, r *http.Request) {
	err := o.stepGitCredentials()
	if err != nil {
		log.Logger().Warn(err.Error())
	}
	arguments := &PipelineRunRequest{}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		o.returnError(err, "could not read the JSON request body: "+err.Error(), w, r)
		return
	}
	err = json.Unmarshal(data, arguments)
	if err != nil {
		o.returnError(err, "failed to unmarshal the JSON request body: "+err.Error(), w, r)
		return
	}
	if err != nil {
		o.returnError(err, "could not parse body: "+err.Error(), w, r)
		return
	}
	log.Logger().Debugf("got payload %#v", arguments)
	pj := arguments.ProwJobSpec

	var revision string
	var prNumber string

	if pj.Refs == nil {
		o.returnError(err, "no prowJobSpec.refs passed in so cannot determine git repository. Input: "+string(data), w, r)
		return
	}

	// Only if there is one Pull in Refs, it's a PR build so we are going to pass it
	if len(pj.Refs.Pulls) == 1 {
		revision = pj.Refs.Pulls[0].SHA
		prNumber = strconv.Itoa(pj.Refs.Pulls[0].Number)
	} else {
		//Otherwise it's a Master / Batch build, and we handle it later
		revision = pj.Refs.BaseSHA
	}

	envs, err := downwardapi.EnvForSpec(downwardapi.NewJobSpec(pj, "", ""))
	if err != nil {
		o.returnError(err, "failed to get env vars from prowjob", w, r)
		return
	}

	sourceURL := fmt.Sprintf("https://github.com/%s/%s.git", pj.Refs.Org, pj.Refs.Repo)
	if sourceURL == "" {
		o.returnError(err, "missing sourceURL property", w, r)
		return
	}
	if revision == "" {
		revision = "master"
	}

	pr := &create.StepCreateTaskOptions{}
	if pj.Type == prowapi.PostsubmitJob {
		pr.PipelineKind = jenkinsfile.PipelineKindRelease
	} else {
		pr.PipelineKind = jenkinsfile.PipelineKindPullRequest
	}

	branch := getBranch(pj)
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
	pr.Context = pj.Context
	pr.Branch = branch
	pr.Revision = revision
	pr.ServiceAccount = o.ServiceAccount

	// turn map into string array with = separator to match type of custom labels which are CLI flags
	for key, value := range arguments.Labels {
		pr.CustomLabels = append(pr.CustomLabels, fmt.Sprintf("%s=%s", key, value))
	}

	// turn map into string array with = separator to match type of custom env vars which are CLI flags
	for key, value := range envs {
		pr.CustomEnvs = append(pr.CustomEnvs, fmt.Sprintf("%s=%s", key, value))
	}

	log.Logger().Infof("triggering pipeline for repo %s branch %s revision %s context %s", sourceURL, branch, revision, pj.Context)

	err = pr.Run()
	if err != nil {
		o.returnError(err, err.Error(), w, r)
		return
	}

	results := &PipelineRunResponse{
		Resources: pr.Results.ObjectReferences(),
	}
	err = o.marshalPayload(w, r, results)
	if err != nil {
		o.returnError(err, "failed to marshal payload", w, r)
	}
	return
}

func (o *ControllerPipelineRunnerOptions) isReady() bool {
	// TODO a better readiness check
	return true
}

func (o *ControllerPipelineRunnerOptions) unmarshalBody(w http.ResponseWriter, r *http.Request, result interface{}) error {
	// TODO assume JSON for now
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "reading the JSON request body")
	}
	err = json.Unmarshal(data, result)
	if err != nil {
		return errors.Wrap(err, "unmarshalling the JSON request body")
	}
	return nil
}

func (o *ControllerPipelineRunnerOptions) marshalPayload(w http.ResponseWriter, r *http.Request, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "marshalling the JSON payload %#v", payload)
	}
	w.Write(data)
	return nil
}

func (o *ControllerPipelineRunnerOptions) onError(err error) {
	if err != nil {
		log.Logger().Errorf("%v", err)
	}
}

func (o *ControllerPipelineRunnerOptions) returnError(err error, message string, w http.ResponseWriter, r *http.Request) {
	log.Logger().Errorf("%v %s", err, message)

	o.onError(err)
	w.WriteHeader(400)
	w.Write([]byte(message))
}

func (o *ControllerPipelineRunnerOptions) stepGitCredentials() error {
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
			return errors.Wrapf(err, "failed to run: jx step gc credentials")
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
