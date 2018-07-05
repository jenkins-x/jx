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

package aggregation

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// we're using /api/v1 right now but aren't doing anything intelligent, if we
	// have an /api/v2 later we'll figure out a good strategy for splitting up the
	// handling.

	// resultsGlobal is the path for node-specific results to be PUT
	resultsByNode = "/api/v1/results/by-node/{node}/{plugin}"
	// resultsGlobal is the path for global (non node-specific) results to be PUT
	resultsGlobal = "/api/v1/results/global/{plugin}"
)

var (
	// Only used for route reversals
	r           = mux.NewRouter()
	nodeRoute   = r.Path(resultsByNode).BuildOnly()
	globalRoute = r.Path(resultsGlobal).BuildOnly()
)

// Handler is a net/http Handler that can handle API requests for aggregation of
// results from nodes, calling the provided callback with the results
type Handler struct {
	mux.Router
	// ResultsCallback is the function that is called when a result is checked in.
	ResultsCallback func(*plugin.Result, http.ResponseWriter)
}

// NewHandler constructs a new aggregation handler which will handler results
// and pass them to the given results callback.
func NewHandler(resultsCallback func(*plugin.Result, http.ResponseWriter)) http.Handler {
	handler := &Handler{
		Router:          *mux.NewRouter(),
		ResultsCallback: resultsCallback,
	}
	// We accept PUT because the client is specifying the resource identifier via
	// the HTTP path. (As opposed to POST, where typically the clients would post
	// to a base URL and the server picks the final resource path.)
	handler.HandleFunc(resultsByNode, handler.resultsHandler).Methods("PUT")
	handler.HandleFunc(resultsGlobal, handler.resultsHandler).Methods("PUT")
	return handler
}

func (h *Handler) resultsHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	vars := mux.Vars(r)

	result := &plugin.Result{
		ResultType: vars["plugin"], // will be empty string in global case
		NodeName:   vars["node"],
		Body:       r.Body,
		MimeType:   r.Header.Get("content-type"),
	}

	// Trigger our callback with this checkin record (which should write the file
	// out.) The callback is responsible for doing a 409 conflict if results are
	// given twice for the same node, etc.
	h.ResultsCallback(result, w)
	r.Body.Close()
}

// NodeResultURL is the URL for results for a given node result. Takes the baseURL (http[s]://hostname:port/,
// with trailing slash) nodeName, pluginName, and an optional extension. If multiple
// extensions are provided, only the first one is used.
func NodeResultURL(baseURL, nodeName, pluginName string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get node result URL")
	}
	path, err := nodeRoute.URLPath("node", nodeName, "plugin", pluginName)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get node result URL")
	}
	path.Scheme = base.Scheme
	path.Host = base.Host
	return path.String(), nil

}

// GlobalResultURL is the URL that results that are not node-specific. Takes the baseURL (http[s]://hostname:port/,
// with trailing slash) pluginName, and an optional extension. If multiple extensions are provided, only the first one
// is used.
func GlobalResultURL(baseURL, pluginName string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get global result URL ")
	}
	path, err := globalRoute.URLPath("plugin", pluginName)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get global result URL ")
	}
	path.Scheme = base.Scheme
	// Host includes port
	path.Host = base.Host
	return path.String(), nil

}

func logRequest(req *http.Request) {
	vars := mux.Vars(req)
	log := logrus.WithField("plugin_name", vars["plugin"])
	if node := vars["node"]; node != "" {
		log = log.WithField("node", node)
	}
	if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
		log = log.WithField("client_cert", req.TLS.PeerCertificates[0].Subject.CommonName)
	}
	log.Info("received aggregator request")
}
