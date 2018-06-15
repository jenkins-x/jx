/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"crypto/tls"
	"io"
	"path"

	"github.com/heptio/sonobuoy/pkg/plugin/manifest"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Interface represents what all plugins must implement to be run and have
// results be aggregated.
type Interface interface {
	// Run runs a plugin, declaring all resources it needs, and then
	// returns.  It does not block and wait until the plugin has finished.
	Run(kubeClient kubernetes.Interface, hostname string, cert *tls.Certificate) error
	// Cleanup cleans up all resources created by the plugin
	Cleanup(kubeClient kubernetes.Interface)
	// Monitor continually checks for problems in the resources created by a
	// plugin (either because it won't schedule, or the image won't
	// download, too many failed executions, etc) and sends the errors as
	// Result objects through the provided channel.
	Monitor(kubeClient kubernetes.Interface, availableNodes []v1.Node, resultsCh chan<- *Result)
	// ExpectedResults is an array of Result objects that a plugin should
	// expect to submit.
	ExpectedResults(nodes []v1.Node) []ExpectedResult
	// FillTemplate fills the driver's internal template so it can be presented to users
	FillTemplate(hostname string, cert *tls.Certificate) ([]byte, error)
	// GetResultType returns the type of results for this plugin, typically
	// the same as the plugin name.
	GetResultType() string
	// GetName returns the name of this plugin
	GetName() string
}

// Definition defines a plugin's features, method of launch, and other
// metadata about it.
type Definition struct {
	Name         string
	ResultType   string
	Spec         manifest.Container
	ExtraVolumes []manifest.Volume
}

// ExpectedResult is an expected result that a plugin will submit.  This is so
// the aggregation server can know when it all results have been received.
type ExpectedResult struct {
	NodeName   string
	ResultType string
}

// Result represents a result we got from a dispatched plugin, returned to the
// aggregation server over HTTP.  Errors running a plugin are also considered a
// Result, if they have an Error property set.
type Result struct {
	NodeName   string
	ResultType string
	MimeType   string
	Body       io.Reader
	Error      string
}

// IsSuccess returns whether the Result represents a successful plugin result,
// versus one that was unsuccessful (for instance, from a dispatched plugin not
// being able to launch.)
func (r *Result) IsSuccess() bool {
	return r.Error == ""
}

// Path is the path within the "plugins" section of the results tarball where
// this Result should be stored, not including a file extension.
func (r *Result) Path() string {
	if !r.IsSuccess() {
		return path.Join(r.ResultType, "errors", r.NodeName)
	}

	return path.Join(r.ResultType, "results", r.NodeName)
}

// Selection is the user specified input to load and initialize plugins
type Selection struct {
	Name string `json:"name"`
}

// AggregationConfig are the config settings for the server that aggregates plugin results
type AggregationConfig struct {
	BindAddress      string `json:"bindaddress"`
	BindPort         int    `json:"bindport"`
	AdvertiseAddress string `json:"advertiseaddress"`
	TimeoutSeconds   int    `json:"timeoutseconds"`
}

// WorkerConfig is the file given to the sonobuoy worker to configure it to phone home.
type WorkerConfig struct {
	// MasterURL is the URL we talk to for submitting results
	MasterURL string `json:"masterurl,omitempty" mapstructure:"masterurl"`
	// NodeName is the node name we should call ourselves when sending results
	NodeName string `json:"nodename,omitempty" mapstructure:"nodename"`
	// ResultsDir is the directory that's expected to contain the host's root filesystem
	ResultsDir string `json:"resultsdir,omitempty" mapstructure:"resultsdir"`
	// ResultType is the type of result (to be put in the HTTP URL's path) to be
	// sent back to sonobuoy.
	ResultType string `json:"resulttype,omitempty" mapstructure:"resulttype"`
	CACert     string `json:"cacert,omitempty" mapstructure:"cacert"`
	ClientCert string `json:"clientcert,omitempty" mapstructure:"clientcert"`
	ClientKey  string `json:"clientkey,omitempty" mapstructure:"clientkey"`
}

// ID returns a unique identifier for this expected result to distinguish it
// from the rest of the results that may be seen by the aggregation server.
func (er *ExpectedResult) ID() string {
	if er.NodeName == "" {
		return er.ResultType
	}

	return er.ResultType + "/" + er.NodeName
}

// ExpectedResultID returns a unique identifier for this result to match it up
// against an expected result.
func (r *Result) ExpectedResultID() string {
	if r.NodeName == "" {
		return r.ResultType
	}

	return r.ResultType + "/" + r.NodeName
}
