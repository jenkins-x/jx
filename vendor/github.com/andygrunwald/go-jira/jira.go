package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/pkg/errors"
)

// A Client manages communication with the JIRA API.
type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests.
	baseURL *url.URL

	// Session storage if the user authentificate with a Session cookie
	session *Session

	// Services used for talking to different parts of the JIRA API.
	Authentication *AuthenticationService
	Issue          *IssueService
	Project        *ProjectService
	Board          *BoardService
	Sprint         *SprintService
	User           *UserService
	Group          *GroupService
	Version        *VersionService
	Priority       *PriorityService
	Field          *FieldService
	Component      *ComponentService
	Resolution     *ResolutionService
	StatusCategory *StatusCategoryService
}

// NewClient returns a new JIRA API client.
// If a nil httpClient is provided, http.DefaultClient will be used.
// To use API methods which require authentication you can follow the preferred solution and
// provide an http.Client that will perform the authentication for you with OAuth and HTTP Basic (such as that provided by the golang.org/x/oauth2 library).
// As an alternative you can use Session Cookie based authentication provided by this package as well.
// See https://docs.atlassian.com/jira/REST/latest/#authentication
// baseURL is the HTTP endpoint of your JIRA instance and should always be specified with a trailing slash.
func NewClient(httpClient *http.Client, baseURL string) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// ensure the baseURL contains a trailing slash so that all paths are preserved in later calls
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		client:  httpClient,
		baseURL: parsedBaseURL,
	}
	c.Authentication = &AuthenticationService{client: c}
	c.Issue = &IssueService{client: c}
	c.Project = &ProjectService{client: c}
	c.Board = &BoardService{client: c}
	c.Sprint = &SprintService{client: c}
	c.User = &UserService{client: c}
	c.Group = &GroupService{client: c}
	c.Version = &VersionService{client: c}
	c.Priority = &PriorityService{client: c}
	c.Field = &FieldService{client: c}
	c.Component = &ComponentService{client: c}
	c.Resolution = &ResolutionService{client: c}
	c.StatusCategory = &StatusCategoryService{client: c}

	return c, nil
}

// NewRawRequest creates an API request.
// A relative URL can be provided in urlStr, in which case it is resolved relative to the baseURL of the Client.
// Allows using an optional native io.Reader for sourcing the request body.
func (c *Client) NewRawRequest(method, urlStr string, body io.Reader) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	// Relative URLs should be specified without a preceding slash since baseURL will have the trailing slash
	rel.Path = strings.TrimLeft(rel.Path, "/")

	u := c.baseURL.ResolveReference(rel)

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Set authentication information
	if c.Authentication.authType == authTypeSession {
		// Set session cookie if there is one
		if c.session != nil {
			for _, cookie := range c.session.Cookies {
				req.AddCookie(cookie)
			}
		}
	} else if c.Authentication.authType == authTypeBasic {
		// Set basic auth information
		if c.Authentication.username != "" {
			req.SetBasicAuth(c.Authentication.username, c.Authentication.password)
		}
	}

	return req, nil
}

// NewRequest creates an API request.
// A relative URL can be provided in urlStr, in which case it is resolved relative to the baseURL of the Client.
// If specified, the value pointed to by body is JSON encoded and included as the request body.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	// Relative URLs should be specified without a preceding slash since baseURL will have the trailing slash
	rel.Path = strings.TrimLeft(rel.Path, "/")

	u := c.baseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Set authentication information
	if c.Authentication.authType == authTypeSession {
		// Set session cookie if there is one
		if c.session != nil {
			for _, cookie := range c.session.Cookies {
				req.AddCookie(cookie)
			}
		}
	} else if c.Authentication.authType == authTypeBasic {
		// Set basic auth information
		if c.Authentication.username != "" {
			req.SetBasicAuth(c.Authentication.username, c.Authentication.password)
		}
	}

	return req, nil
}

// addOptions adds the parameters in opt as URL query parameters to s.  opt
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// NewMultiPartRequest creates an API request including a multi-part file.
// A relative URL can be provided in urlStr, in which case it is resolved relative to the baseURL of the Client.
// If specified, the value pointed to by buf is a multipart form.
func (c *Client) NewMultiPartRequest(method, urlStr string, buf *bytes.Buffer) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	// Relative URLs should be specified without a preceding slash since baseURL will have the trailing slash
	rel.Path = strings.TrimLeft(rel.Path, "/")

	u := c.baseURL.ResolveReference(rel)

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	// Set required headers
	req.Header.Set("X-Atlassian-Token", "nocheck")

	// Set authentication information
	if c.Authentication.authType == authTypeSession {
		// Set session cookie if there is one
		if c.session != nil {
			for _, cookie := range c.session.Cookies {
				req.AddCookie(cookie)
			}
		}
	} else if c.Authentication.authType == authTypeBasic {
		// Set basic auth information
		if c.Authentication.username != "" {
			req.SetBasicAuth(c.Authentication.username, c.Authentication.password)
		}
	}

	return req, nil
}

// Do sends an API request and returns the API response.
// The API response is JSON decoded and stored in the value pointed to by v, or returned as an error if an API error has occurred.
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	err = CheckResponse(httpResp)
	if err != nil {
		// Even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return newResponse(httpResp, nil), err
	}

	if v != nil {
		// Open a NewDecoder and defer closing the reader only if there is a provided interface to decode to
		defer httpResp.Body.Close()
		err = json.NewDecoder(httpResp.Body).Decode(v)
	}

	resp := newResponse(httpResp, v)
	return resp, err
}

// CheckResponse checks the API response for errors, and returns them if present.
// A response is considered an error if it has a status code outside the 200 range.
// The caller is responsible to analyze the response body.
// The body can contain JSON (if the error is intended) or xml (sometimes JIRA just failes).
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	err := fmt.Errorf("Request failed. Please analyze the request body for more details. Status code: %d", r.StatusCode)
	return err
}

// GetBaseURL will return you the Base URL.
// This is the same URL as in the NewClient constructor
func (c *Client) GetBaseURL() url.URL {
	return *c.baseURL
}

// Response represents JIRA API response. It wraps http.Response returned from
// API and provides information about paging.
type Response struct {
	*http.Response

	StartAt    int
	MaxResults int
	Total      int
}

func newResponse(r *http.Response, v interface{}) *Response {
	resp := &Response{Response: r}
	resp.populatePageValues(v)
	return resp
}

// Sets paging values if response json was parsed to searchResult type
// (can be extended with other types if they also need paging info)
func (r *Response) populatePageValues(v interface{}) {
	switch value := v.(type) {
	case *searchResult:
		r.StartAt = value.StartAt
		r.MaxResults = value.MaxResults
		r.Total = value.Total
	case *groupMembersResult:
		r.StartAt = value.StartAt
		r.MaxResults = value.MaxResults
		r.Total = value.Total
	}
	return
}

// BasicAuthTransport is an http.RoundTripper that authenticates all requests
// using HTTP Basic Authentication with the provided username and password.
type BasicAuthTransport struct {
	Username string
	Password string

	// Transport is the underlying HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	Transport http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.  We just add the
// basic auth and return the RoundTripper for this transport type.
func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := cloneRequest(req) // per RoundTripper contract

	req2.SetBasicAuth(t.Username, t.Password)
	return t.transport().RoundTrip(req2)
}

// Client returns an *http.Client that makes requests that are authenticated
// using HTTP Basic Authentication.  This is a nice little bit of sugar
// so we can just get the client instead of creating the client in the calling code.
// If it's necessary to send more information on client init, the calling code can
// always skip this and set the transport itself.
func (t *BasicAuthTransport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func (t *BasicAuthTransport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

// CookieAuthTransport is an http.RoundTripper that authenticates all requests
// using Jira's cookie-based authentication.
//
// Note that it is generally preferrable to use HTTP BASIC authentication with the REST API.
// However, this resource may be used to mimic the behaviour of JIRA's log-in page (e.g. to display log-in errors to a user).
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/latest/#auth/1/session
type CookieAuthTransport struct {
	Username string
	Password string
	AuthURL  string

	// SessionObject is the authenticated cookie string.s
	// It's passed in each call to prove the client is authenticated.
	SessionObject []*http.Cookie

	// Transport is the underlying HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	Transport http.RoundTripper
}

// RoundTrip adds the session object to the request.
func (t *CookieAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.SessionObject == nil {
		err := t.setSessionObject()
		if err != nil {
			return nil, errors.Wrap(err, "cookieauth: no session object has been set")
		}
	}

	req2 := cloneRequest(req) // per RoundTripper contract
	for _, cookie := range t.SessionObject {
		req2.AddCookie(cookie)
	}

	return t.transport().RoundTrip(req2)
}

// Client returns an *http.Client that makes requests that are authenticated
// using cookie authentication
func (t *CookieAuthTransport) Client() *http.Client {
	return &http.Client{Transport: t}
}

// setSessionObject attempts to authenticate the user and set
// the session object (e.g. cookie)
func (t *CookieAuthTransport) setSessionObject() error {
	req, err := t.buildAuthRequest()
	if err != nil {
		return err
	}

	var authClient = &http.Client{
		Timeout: time.Second * 60,
	}
	resp, err := authClient.Do(req)
	if err != nil {
		return err
	}

	t.SessionObject = resp.Cookies()
	return nil
}

// getAuthRequest assembles the request to get the authenticated cookie
func (t *CookieAuthTransport) buildAuthRequest() (*http.Request, error) {
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		t.Username,
		t.Password,
	}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(body)

	req, err := http.NewRequest("POST", t.AuthURL, b)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (t *CookieAuthTransport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}
