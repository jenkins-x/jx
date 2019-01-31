package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	pipelineapi "github.com/knative/build-pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"
)

// ControllerPipelineRunnerOptions holds the command line arguments
type ControllerPipelineRunnerOptions struct {
	CommonOptions
	BindAddress string
	Path        string
	Port        int
}

// PipelineRunRequest the request to trigger a pipeline run
type PipelineRunRequest struct {
	GitURL  string `json:"gitUrl,omitempty"`
	Branch  string `json:"branch,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Context string `json:"context,omitempty"`
}

// PipelineRunResponse the results of triggering a pipeline run
type PipelineRunResponse struct {
}

var (
	controllerPipelineRunnersLong = templates.LongDesc(`Runs the service to generate Knative PipelineRun resources from source code webhooks`)

	controllerPipelineRunnersExample = templates.Examples(`
			# run the pipeline runner controller
			jx controller pipelinerunner
		`)
)

// NewCmdControllerPipelineRunner creates the command
func NewCmdControllerPipelineRunner(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := ControllerPipelineRunnerOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "pipelinerunner",
		Short:   "Runs the service to generate Knative PipelineRun resources from source code webhooks",
		Long:    controllerPipelineRunnersLong,
		Example: controllerPipelineRunnersExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)

	cmd.Flags().IntVarP(&options.Port, optionPort, "", 8080, "The TCP port to listen on.")
	cmd.Flags().StringVarP(&options.BindAddress, optionBind, "", "",
		"The interface address to bind to (by default, will listen on all interfaces/addresses).")
	cmd.Flags().StringVarP(&options.Path, "path", "p", "/",
		"The path to listen on for requests to trigger a pipeline run.")
	return cmd
}

// Run will implement this command
func (o *ControllerPipelineRunnerOptions) Run() error {
	mux := http.NewServeMux()
	mux.Handle(o.Path, http.HandlerFunc(o.piplineRunMethods))
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))

	logrus.Infof("Serving build numbers at http://%s:%d%s", o.BindAddress, o.Port, o.Path)
	return http.ListenAndServe(":"+strconv.Itoa(o.Port), mux)
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *ControllerPipelineRunnerOptions) health(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *ControllerPipelineRunnerOptions) ready(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// handle request for pipeline runs
func (o *ControllerPipelineRunnerOptions) piplineRunMethods(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		fmt.Fprintf(w, "Please POST JSON to this endpoint!\n")
	case http.MethodHead:
		logrus.Info("HEAD Todo...")
	case http.MethodPost:
		o.startPipelineRun(w, r)
	default:
		logrus.Errorf("Unsupported method %s for %s", r.Method, o.Path)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

// handle request for pipeline runs
func (o *ControllerPipelineRunnerOptions) startPipelineRun(w http.ResponseWriter, r *http.Request) {
	arguments := &PipelineRunRequest{}
	err := o.unmarshalBody(w, r, arguments)
	o.onError(err)
	if err != nil {
		return
	}
	if o.Verbose {
		logrus.Infof("got payload %#v", arguments)
	}

	if arguments.GitURL == "" {
		o.returnError("missing gitUrl property", w, r)
		return
	}
	if arguments.Branch == "" {
		arguments.Branch = "master"
	}
	if arguments.Kind == "" {
		arguments.Kind = "release"
	}


	pr := &StepCreateTaskOptions{}
	pr.CommonOptions = o.CommonOptions

	// defaults
	pr.SourceName = "source"
	pr.Duration = time.Second * 20
	pr.Trigger = string(pipelineapi.PipelineTriggerTypeManual)

	pr.CloneGitURL = arguments.GitURL
	pr.DeleteTempDir = true
	pr.Context = arguments.Context
	pr.Branch = arguments.Branch
	pr.PipelineKind = arguments.Kind

	err = pr.Run()
	if err != nil {
		o.returnError(err.Error(), w, r)
		return
	}

	results := &PipelineRunResponse{}
	err = o.marshalPayload(w, r, results)
	o.onError(err)
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
		logrus.Errorf("%v", err)
	}
}

func (o *ControllerPipelineRunnerOptions) returnError(message string, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(400)
	w.Write([]byte(message))
}
