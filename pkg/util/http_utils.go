package util

import (
	"github.com/jenkins-x/jx/pkg/log"
	"net"
	"net/http"
	. "os"
	"strconv"
	"time"
)

// defaults mirror the default http.Transport values
var jxDefaultTransport http.RoundTripper = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   time.Duration(getIntFromEnv("HTTP_DIALER_TIMEOUT", 30)) * time.Second,
		KeepAlive: time.Duration(getIntFromEnv("HTTP_DIALER_KEEP_ALIVE", 30)) * time.Second,
		DualStack: getBoolFromEnv("HTTP_USE_DUAL_STACK", true),
	}).DialContext,
	MaxIdleConns:          getIntFromEnv("HTTP_MAX_IDLE_CONNS", 100),
	IdleConnTimeout:       time.Duration(getIntFromEnv("HTTP_IDLE_CONN_TIMEOUT", 90)) * time.Second,
	TLSHandshakeTimeout:   time.Duration(getIntFromEnv("HTTP_TLS_HANDSHAKE_TIMEOUT", 10)) * time.Second,
	ExpectContinueTimeout: time.Duration(getIntFromEnv("HTTP_EXPECT_CONTINUE_TIMEOUT", 1)) * time.Second,
}

var defaultClient = http.Client{Transport: jxDefaultTransport, Timeout: time.Duration(getIntFromEnv("DEFAULT_HTTP_REQUEST_TIMEOUT", 30)) * time.Second}

// returns a Client reference with our default configuration
func GetClient() (*http.Client) {
	return &defaultClient
}

// returns a client with JX default transport and user specified timeout (in seconds)
func GetClientWithTimeout(timeout int) (*http.Client) {
	client := http.Client{}
	client.Transport = jxDefaultTransport
	client.Timeout = time.Duration(timeout) * time.Second
	return &client
}

// returns a client with user specified transport and timeout (in seconds)
func GetCustomClient(transport *http.Transport, timeout int) (*http.Client) {
	return &(http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Second})
}

func getIntFromEnv(key string, fallback int) (int) {
	if value, ok := LookupEnv(key); ok {
		int_value, err := strconv.Atoi(value)
		if err == nil {
			return int_value
		} else {
			log.Warnf("Unable to convert env var %s with value %s to integer, using default value of %s instead", key, value, fallback)
		}
	}
	return fallback
}

func getBoolFromEnv(key string, fallback bool) bool {
	if value, ok := LookupEnv(key); ok {
		bool_value, err := strconv.ParseBool(value)
		if err == nil {
			return bool_value
		} else {
			log.Warnf("Unable to convert env var %s with value %s to boolean, using default value of %s instead", key, value, fallback)
		}
	}
	return fallback
}
