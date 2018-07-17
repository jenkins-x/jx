package codeship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// ErrRateLimitExceeded occurs when Codeship returns 403 Forbidden response
var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// ErrNotFound occurs when Codeship returns a 404 Not Found response
type ErrNotFound struct {
	apiErrors
}

// ErrBadRequest occurs when Codeship returns a 400 Bad Request response
type ErrBadRequest struct {
	apiErrors
}

type apiErrors struct {
	Errors []string `json:"errors"`
}

func (e apiErrors) Error() string {
	return strings.Join(e.Errors, ", ")
}

// Organization holds the configuration for the current API client scoped to the Organization. Should not
// be modified concurrently
type Organization struct {
	UUID   string
	Name   string
	Scopes []string
	client *Client
}

// Response is a Codeship response. This wraps the standard http.Response returned from Codeship.
type Response struct {
	*http.Response
	// Links that were returned with the response. These are parsed from the Link header.
	Links
}

var (
	urlRegex = regexp.MustCompile(`\s*<(.+)>`)
	relRegex = regexp.MustCompile(`\s*rel="(\w+)"`)
)

func newResponse(r *http.Response) Response {
	response := Response{Response: r}

	if linkText := r.Header.Get("Link"); linkText != "" {
		linkMap := make(map[string]string)

		// one chunk: <url>; rel="foo"
		for _, chunk := range strings.Split(linkText, ",") {
			pieces := strings.Split(chunk, ";")
			urlMatch := urlRegex.FindStringSubmatch(pieces[0])
			relMatch := relRegex.FindStringSubmatch(pieces[1])

			if len(relMatch) > 1 && len(urlMatch) > 1 {
				linkMap[relMatch[1]] = urlMatch[1]
			}
		}

		response.Links.First = linkMap["first"]
		response.Links.Last = linkMap["last"]
		response.Links.Next = linkMap["next"]
		response.Links.Previous = linkMap["prev"]
	}

	return response
}

const apiURL = "https://api.codeship.com/v2"

// Client holds information necessary to make a request to the Codeship API
type Client struct {
	baseURL        string
	authenticator  Authenticator
	authentication Authentication
	headers        http.Header
	httpClient     *http.Client
	logger         *log.Logger
	verbose        bool
}

// New creates a new Codeship API client
func New(auth Authenticator, opts ...Option) (*Client, error) {
	if auth == nil {
		return nil, errors.New("no authenticator provided")
	}

	client := &Client{
		authenticator: auth,
		baseURL:       apiURL,
		headers:       make(http.Header),
	}

	if err := client.parseOptions(opts...); err != nil {
		return nil, errors.Wrap(err, "options parsing failed")
	}

	// Fall back to http.DefaultClient if the user does not provide
	// their own
	if client.httpClient == nil {
		client.httpClient = &http.Client{
			Timeout: time.Second * 30,
		}
	}

	// Fall back to default log.Logger (STDOUT) if the user does not provide
	// their own
	if client.logger == nil {
		client.logger = &log.Logger{}
		client.logger.SetOutput(os.Stdout)
	}

	return client, nil
}

// Organization scopes a client to a single Organization, allowing the user to make calls to the API
func (c *Client) Organization(ctx context.Context, name string) (*Organization, error) {
	if name == "" {
		return nil, errors.New("no organization name provided")
	}

	if c.AuthenticationRequired() {
		if _, err := c.Authenticate(ctx); err != nil {
			return nil, errors.Wrap(err, "authentication failed")
		}
	}

	for _, org := range c.authentication.Organizations {
		if org.Name == strings.ToLower(name) {
			return &Organization{
				UUID:   org.UUID,
				Name:   org.Name,
				Scopes: org.Scopes,
				client: c,
			}, nil
		}
	}
	return nil, ErrUnauthorized(fmt.Sprintf("organization %q not authorized. Authorized organizations: %v", name, c.authentication.Organizations))
}

// Authentication returns the client's current Authentication object
func (c *Client) Authentication() Authentication {
	return c.authentication
}

// AuthenticationRequired determines if a client must authenticate before making a request
func (c *Client) AuthenticationRequired() bool {
	return c.authentication.AccessToken == "" || c.authentication.ExpiresAt <= time.Now().Unix()
}

func (c *Client) request(ctx context.Context, method, path string, params interface{}) ([]byte, Response, error) {
	url := c.baseURL + path
	// Replace nil with a JSON object if needed
	var reqBody io.Reader
	if params != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(params); err != nil {
			return nil, Response{}, err
		}
		reqBody = buf
	}

	if c.AuthenticationRequired() {
		if _, err := c.Authenticate(ctx); err != nil {
			return nil, Response{}, err
		}
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, Response{}, errors.Wrap(err, "HTTP request creation failed")
	}

	// Apply any user-defined headers first
	req.Header = cloneHeader(c.headers)
	req.Header.Set("Authorization", "Bearer "+c.authentication.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.do(req.WithContext(ctx))
}

func (c *Client) do(req *http.Request) ([]byte, Response, error) {
	if c.verbose {
		dumpReq, _ := httputil.DumpRequest(req, true)
		c.logger.Println(string(dumpReq))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, Response{}, errors.Wrap(err, "HTTP request failed")
	}

	if c.verbose {
		dumpResp, _ := httputil.DumpResponse(resp, true)
		c.logger.Println(string(dumpResp))
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	response := newResponse(resp)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, response, errors.Wrap(err, "could not read response body")
	}

	code := resp.StatusCode
	if code < 400 {
		return body, response, nil
	}

	switch code {
	case http.StatusBadRequest:
		var e ErrBadRequest
		if err = json.Unmarshal(body, &e); err != nil {
			return nil, response, ErrBadRequest{}
		}
		return nil, response, e
	case http.StatusNotFound:
		var e ErrNotFound
		if err = json.Unmarshal(body, &e); err != nil {
			return nil, response, ErrNotFound{}
		}
		return nil, response, e
	case http.StatusUnauthorized:
		return nil, response, ErrUnauthorized("invalid credentials")
	case http.StatusForbidden, http.StatusTooManyRequests:
		return nil, response, ErrRateLimitExceeded
	}

	if len(body) > 0 {
		return nil, response, fmt.Errorf("HTTP status: %d; content %q", resp.StatusCode, string(body))
	}
	return nil, response, fmt.Errorf("HTTP status: %d", resp.StatusCode)
}

// cloneHeader returns a shallow copy of the header.
// copied from https://godoc.org/github.com/golang/gddo/httputil/header#Copy
func cloneHeader(header http.Header) http.Header {
	h := make(http.Header)
	for k, vs := range header {
		h[k] = vs
	}
	return h
}
