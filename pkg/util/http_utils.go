package util

import (
	"net"
	"net/http"
	"time"
)

// currently mirrors the default http.Transport values
var JxDefaultTransport http.RoundTripper = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var DEFAULT_HTTP_REQUEST_TIMEOUT = 30
var isInitialized bool = false
var defaultClient = http.Client{}

// returns a Client reference with our default configuration
func GetClient() (*http.Client) {
	if (!isInitialized) {
		// initialize client with our default values
		defaultClient.Transport = JxDefaultTransport
		defaultClient.Timeout = time.Duration(DEFAULT_HTTP_REQUEST_TIMEOUT) * time.Second
		isInitialized = true
	}
	return &defaultClient
}

// returns a client with JX default transport and user specified timeout (in seconds)
func GetClientWithTimeout(timeout int) (*http.Client){
	client := http.Client{}
	client.Transport = JxDefaultTransport
	client.Timeout = time.Duration(timeout) * time.Second
	return &client
}

// returns a client with user specified transport and timeout (in seconds)
func GetCustomClient(transport *http.Transport, timeout int) (*http.Client) {
	return &(http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Second})
}

