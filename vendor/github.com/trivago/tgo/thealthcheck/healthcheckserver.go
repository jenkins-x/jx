// This package provides a simple health check server.
//
// The package lets you register arbitrary health check endpoints by
// providing a name (== URL path) and a callback function.
//
// The health check server listens for HTTP requests on a given port;
// probes are executed by issuing requests to the endpoints' named
// URL paths.
//
// GETing the "/" path provides a list of registered endpoints, one
// per line. GETing "/_ALL_" probes each registered endpoint sequentially,
// returning each endpoint's path, HTTP status code and body per line.
//
// The package works as a "singleton" with just one server in order to
// avoid cluttering the main program by passing handles around.
package thealthcheck

import (
	"net/http"
	"fmt"
	"bytes"
)

//
const (
	StatusOK = http.StatusOK
	StatusServiceUnavailable = http.StatusServiceUnavailable
)

// Code wishing to get probed by the health-checker needs to provide this callback
type CallbackFunc func () (code int, body string)

// The HTTP server
var server *http.Server

// The HTTP request multiplexer
var serveMux *http.ServeMux

// List of endpoints known by the server
var endpoints map[string]CallbackFunc

// Init
func init() {
	// Create the request multiplexer
	serveMux = http.NewServeMux()
	// Initialize the endpoint list
	endpoints = make(map[string]CallbackFunc)
}

// Configures the health check server
//
//  listenAddr: an address understood by http.ListenAndServe(), e.g. ":8008"
func Configure(listenAddr string) {
	// Create the HTTP server
	server = &http.Server{
		Addr:    listenAddr,
		Handler: serveMux,
	}

	// Add default wildcard handler
	serveMux.HandleFunc("/",
		func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
			path := httpRequest.URL.Path

			// Handle "/": list all our registered endpoints
			if path == "/" {
				fmt.Fprintf(responseWriter, "/_ALL_\n")

				// TBD: Collate
				for endpointPath, _ := range endpoints {
					fmt.Fprintf(responseWriter, "%s\n", endpointPath)
				}
				return
			}

			// Default action: 404
			responseWriter.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(responseWriter, "Path not found\n")
			return
		},
	)

	// Add magical "/_ALL" handler
	serveMux.HandleFunc("/_ALL_",
		func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
			// Response code needs to be set before writing the response body,
			// so we need to pool the body temporarily into resultBody
			var resultCode = StatusOK
			var resultBody bytes.Buffer

			// Call all endpoints sequentially
			// (TBD: if there were _a lot_ of these we could do them in parallel)
			// TBD: Collate (output at least)
			for endpointPath, callback := range endpoints {
				// Call the callback
				code, body := callback()
				// Append path, code, body to response body
				fmt.Fprintf(&resultBody,
					"%s %d %s\n",
					endpointPath,
					code,
					body,
				)

				if code != StatusOK {
					resultCode = StatusServiceUnavailable
				}
			}

			// Set HTTP response code
			responseWriter.WriteHeader(resultCode)

			// Write HTTP response body
			// (TBD: more efficient way, buffer -> writer?)
			fmt.Fprintf(responseWriter, resultBody.String())
		},
	)
}

// Registers an endpoint with the health checker.
//
// The urlPath must be unique. The callback must return an HTTP response code
// and body text.
//
// Boilerplate:
//
//  healthcheck.AddEndpoint("/my/arbitrary/path" func()(code int, body string) {
//      return 200, "Foobar Plugin is OK"
//  })
//
func AddEndpoint(urlPath string, callback CallbackFunc){
	// Check parameters
	// -syntax
	if len(urlPath) == 0 || urlPath[:1] != "/" || urlPath[len(urlPath)-1:] == "/" {
		panic(fmt.Sprintf(
			"ERROR: Health check endpoint must begin and may not end with a slash: \"%s\"",
			urlPath))
	}
	// - reserved paths
	for _, path := range []string{"/", "/_ALL_"} {
		if urlPath == path {
			panic(fmt.Sprintf(
				"ERROR: Health check path \"%s\" is reserved", path))
		}
	}
	// - registered paths
	_, exists := endpoints[urlPath]
	if exists {
		panic(fmt.Sprintf(
			"ERROR: Health check endpoint \"%s\" already registered", urlPath))
	}

	// Register the HTTP route & handler
	serveMux.HandleFunc(
		urlPath,
		func(responseWriter http.ResponseWriter, httpRequest *http.Request){
			// Call the callback
			code, body := callback()
			// Set HTTP response code
			responseWriter.WriteHeader(code)
			// Write HTTP response body
			fmt.Fprintf(responseWriter, body)
			fmt.Fprintf(responseWriter, "\n")
		},
	)

	// Store the endpoint
	endpoints[urlPath] = callback
}

// Registers an endpoint with the health checker.
//
// This is a convenience version of AddEndpoint() that takes
// the urlPath's components as a list of strings and catenates
// them.
func AddEndpointPathArray(urlPath []string, callback CallbackFunc) {
	// Catenate path
	var cat bytes.Buffer
	for _, pathComponent := range urlPath {
		fmt.Fprintf(&cat, "/%s", pathComponent)
	}
	// Call it
	AddEndpoint(cat.String(), callback)
}

// Starts the HTTP server
//
// Call this after Configure() and AddEndpoint() calls.
//
// TBD: is it possible to AddEndpoint() after Start()ing?
func Start(){
	err := server.ListenAndServe()

	if err != nil {
		panic(err.Error())
	}
}

// TBD: Cleanup?
func Stop(){


}
