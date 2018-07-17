package codeship

import (
	"log"
	"net/http"
)

// Option is a functional option for configuring the API client
type Option func(*Client) error

// HTTPClient accepts a custom *http.Client for making API calls
func HTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = client
		return nil
	}
}

// Headers allows you to set custom HTTP headers when making API calls (e.g. for
// satisfying HTTP proxies, or for debugging)
func Headers(headers http.Header) Option {
	return func(c *Client) error {
		c.headers = headers
		return nil
	}
}

// BaseURL allows overriding of API client baseURL for testing
func BaseURL(baseURL string) Option {
	return func(c *Client) error {
		c.baseURL = baseURL
		return nil
	}
}

// Logger allows overriding the default STDOUT logger
func Logger(logger *log.Logger) Option {
	return func(c *Client) error {
		c.logger = logger
		return nil
	}
}

// Verbose allows enabling/disabling internal logging
func Verbose(verbose bool) Option {
	return func(c *Client) error {
		c.verbose = verbose
		return nil
	}
}

// parseOptions parses the supplied options functions and returns a configured
// *Client instance
func (c *Client) parseOptions(opts ...Option) error {
	// Range over each options function and apply it to our API type to
	// configure it. Options functions are applied in order, with any
	// conflicting options overriding earlier calls.
	for _, option := range opts {
		err := option(c)
		if err != nil {
			return err
		}
	}

	return nil
}
