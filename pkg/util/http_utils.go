package util

import (
	"net"
	"net/http"
	"time"
)

// currently mirrors the default http.Transport values
var jxDefaultTransport http.RoundTripper = &http.Transport{
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

const DEFAULT_HTTP_REQUEST_TIMEOUT = 30
var defaultClient = http.Client{Transport: jxDefaultTransport, Timeout: time.Duration(DEFAULT_HTTP_REQUEST_TIMEOUT) * time.Second}

// returns a Client reference with our default configuration
func GetClient() (*http.Client) {
	return &defaultClient
}

// returns a client with JX default transport and user specified timeout (in seconds)
func GetClientWithTimeout(timeout int) (*http.Client){
	client := http.Client{}
	client.Transport = jxDefaultTransport
	client.Timeout = time.Duration(timeout) * time.Second
	return &client
}

// returns a client with user specified transport and timeout (in seconds)
func GetCustomClient(transport *http.Transport, timeout int) (*http.Client) {
	return &(http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Second})
}

