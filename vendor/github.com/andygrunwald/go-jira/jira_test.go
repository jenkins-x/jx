package jira

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	testJIRAInstanceURL = "https://issues.apache.org/jira/"
)

var (
	// testMux is the HTTP request multiplexer used with the test server.
	testMux *http.ServeMux

	// testClient is the JIRA client being tested.
	testClient *Client

	// testServer is a test HTTP server used to provide mock API responses.
	testServer *httptest.Server
)

type testValues map[string]string

// setup sets up a test HTTP server along with a jira.Client that is configured to talk to that test server.
// Tests should register handlers on mux which provide mock responses for the API method being tested.
func setup() {
	// Test server
	testMux = http.NewServeMux()
	testServer = httptest.NewServer(testMux)

	// jira client configured to use test server
	testClient, _ = NewClient(nil, testServer.URL)
}

// teardown closes the test HTTP server.
func teardown() {
	testServer.Close()
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func testRequestURL(t *testing.T, r *http.Request, want string) {
	if got := r.URL.String(); !strings.HasPrefix(got, want) {
		t.Errorf("Request URL: %v, want %v", got, want)
	}
}

func TestNewClient_WrongUrl(t *testing.T) {
	c, err := NewClient(nil, "://issues.apache.org/jira/")

	if err == nil {
		t.Error("Expected an error. Got none")
	}
	if c != nil {
		t.Errorf("Expected no client. Got %+v", c)
	}
}

func TestNewClient_WithHttpClient(t *testing.T) {
	httpClient := http.DefaultClient
	httpClient.Timeout = 10 * time.Minute
	c, err := NewClient(httpClient, testJIRAInstanceURL)

	if err != nil {
		t.Errorf("Got an error: %s", err)
	}
	if c == nil {
		t.Error("Expected a client. Got none")
	}
	if !reflect.DeepEqual(c.client, httpClient) {
		t.Errorf("HTTP clients are not equal. Injected %+v, got %+v", httpClient, c.client)
	}
}

func TestNewClient_WithServices(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)

	if err != nil {
		t.Errorf("Got an error: %s", err)
	}
	if c.Authentication == nil {
		t.Error("No AuthenticationService provided")
	}
	if c.Issue == nil {
		t.Error("No IssueService provided")
	}
	if c.Project == nil {
		t.Error("No ProjectService provided")
	}
	if c.Board == nil {
		t.Error("No BoardService provided")
	}
	if c.Sprint == nil {
		t.Error("No SprintService provided")
	}
	if c.User == nil {
		t.Error("No UserService provided")
	}
	if c.Group == nil {
		t.Error("No GroupService provided")
	}
	if c.Version == nil {
		t.Error("No VersionService provided")
	}
}

func TestCheckResponse(t *testing.T) {
	codes := []int{
		http.StatusOK, http.StatusPartialContent, 299,
	}

	for _, c := range codes {
		r := &http.Response{
			StatusCode: c,
		}
		if err := CheckResponse(r); err != nil {
			t.Errorf("CheckResponse throws an error: %s", err)
		}
	}
}

func TestClient_NewRequest(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	inURL, outURL := "rest/api/2/issue/", testJIRAInstanceURL+"rest/api/2/issue/"
	inBody, outBody := &Issue{Key: "MESOS"}, `{"key":"MESOS"}`+"\n"
	req, _ := c.NewRequest("GET", inURL, inBody)

	// Test that relative URL was expanded
	if got, want := req.URL.String(), outURL; got != want {
		t.Errorf("NewRequest(%q) URL is %v, want %v", inURL, got, want)
	}

	// Test that body was JSON encoded
	body, _ := ioutil.ReadAll(req.Body)
	if got, want := string(body), outBody; got != want {
		t.Errorf("NewRequest(%v) Body is %v, want %v", inBody, got, want)
	}
}

func TestClient_NewRawRequest(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	inURL, outURL := "rest/api/2/issue/", testJIRAInstanceURL+"rest/api/2/issue/"

	outBody := `{"key":"MESOS"}` + "\n"
	inBody := outBody
	req, _ := c.NewRawRequest("GET", inURL, strings.NewReader(outBody))

	// Test that relative URL was expanded
	if got, want := req.URL.String(), outURL; got != want {
		t.Errorf("NewRawRequest(%q) URL is %v, want %v", inURL, got, want)
	}

	// Test that body was JSON encoded
	body, _ := ioutil.ReadAll(req.Body)
	if got, want := string(body), outBody; got != want {
		t.Errorf("NewRawRequest(%v) Body is %v, want %v", inBody, got, want)
	}
}

func testURLParseError(t *testing.T, err error) {
	if err == nil {
		t.Errorf("Expected error to be returned")
	}
	if err, ok := err.(*url.Error); !ok || err.Op != "parse" {
		t.Errorf("Expected URL parse error, got %+v", err)
	}
}

func TestClient_NewRequest_BadURL(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}
	_, err = c.NewRequest("GET", ":", nil)
	testURLParseError(t, err)
}

func TestClient_NewRequest_SessionCookies(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	cookie := &http.Cookie{Name: "testcookie", Value: "testvalue"}
	c.session = &Session{Cookies: []*http.Cookie{cookie}}
	c.Authentication.authType = authTypeSession

	inURL := "rest/api/2/issue/"
	inBody := &Issue{Key: "MESOS"}
	req, err := c.NewRequest("GET", inURL, inBody)

	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	if len(req.Cookies()) != len(c.session.Cookies) {
		t.Errorf("An error occurred. Expected %d cookie(s). Got %d.", len(c.session.Cookies), len(req.Cookies()))
	}

	for i, v := range req.Cookies() {
		if v.String() != c.session.Cookies[i].String() {
			t.Errorf("An error occurred. Unexpected cookie. Expected %s, actual %s.", v.String(), c.session.Cookies[i].String())
		}
	}
}

func TestClient_NewRequest_BasicAuth(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	c.Authentication.SetBasicAuth("test-user", "test-password")

	inURL := "rest/api/2/issue/"
	inBody := &Issue{Key: "MESOS"}
	req, err := c.NewRequest("GET", inURL, inBody)

	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	username, password, ok := req.BasicAuth()
	if !ok || username != "test-user" || password != "test-password" {
		t.Errorf("An error occurred. Expected basic auth username %s and password %s. Got username %s and password %s.", "test-user", "test-password", username, password)
	}
}

// If a nil body is passed to gerrit.NewRequest, make sure that nil is also passed to http.NewRequest.
// In most cases, passing an io.Reader that returns no content is fine,
// since there is no difference between an HTTP request body that is an empty string versus one that is not set at all.
// However in certain cases, intermediate systems may treat these differently resulting in subtle errors.
func TestClient_NewRequest_EmptyBody(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}
	req, err := c.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest returned unexpected error: %v", err)
	}
	if req.Body != nil {
		t.Fatalf("constructed request contains a non-nil Body")
	}
}

func TestClient_NewMultiPartRequest(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	cookie := &http.Cookie{Name: "testcookie", Value: "testvalue"}
	c.session = &Session{Cookies: []*http.Cookie{cookie}}
	c.Authentication.authType = authTypeSession

	inURL := "rest/api/2/issue/"
	inBuf := bytes.NewBufferString("teststring")
	req, err := c.NewMultiPartRequest("GET", inURL, inBuf)

	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	if len(req.Cookies()) != len(c.session.Cookies) {
		t.Errorf("An error occurred. Expected %d cookie(s). Got %d.", len(c.session.Cookies), len(req.Cookies()))
	}

	for i, v := range req.Cookies() {
		if v.String() != c.session.Cookies[i].String() {
			t.Errorf("An error occurred. Unexpected cookie. Expected %s, actual %s.", v.String(), c.session.Cookies[i].String())
		}
	}

	if req.Header.Get("X-Atlassian-Token") != "nocheck" {
		t.Errorf("An error occurred. Unexpected X-Atlassian-Token header value. Expected nocheck, actual %s.", req.Header.Get("X-Atlassian-Token"))
	}
}

func TestClient_NewMultiPartRequest_BasicAuth(t *testing.T) {
	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	c.Authentication.SetBasicAuth("test-user", "test-password")

	inURL := "rest/api/2/issue/"
	inBuf := bytes.NewBufferString("teststring")
	req, err := c.NewMultiPartRequest("GET", inURL, inBuf)

	if err != nil {
		t.Errorf("An error occurred. Expected nil. Got %+v.", err)
	}

	username, password, ok := req.BasicAuth()
	if !ok || username != "test-user" || password != "test-password" {
		t.Errorf("An error occurred. Expected basic auth username %s and password %s. Got username %s and password %s.", "test-user", "test-password", username, password)
	}

	if req.Header.Get("X-Atlassian-Token") != "nocheck" {
		t.Errorf("An error occurred. Unexpected X-Atlassian-Token header value. Expected nocheck, actual %s.", req.Header.Get("X-Atlassian-Token"))
	}
}

func TestClient_Do(t *testing.T) {
	setup()
	defer teardown()

	type foo struct {
		A string
	}

	testMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m := "GET"; m != r.Method {
			t.Errorf("Request method = %v, want %v", r.Method, m)
		}
		fmt.Fprint(w, `{"A":"a"}`)
	})

	req, _ := testClient.NewRequest("GET", "/", nil)
	body := new(foo)
	testClient.Do(req, body)

	want := &foo{"a"}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("Response body = %v, want %v", body, want)
	}
}

func TestClient_Do_HTTPResponse(t *testing.T) {
	setup()
	defer teardown()

	type foo struct {
		A string
	}

	testMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m := "GET"; m != r.Method {
			t.Errorf("Request method = %v, want %v", r.Method, m)
		}
		fmt.Fprint(w, `{"A":"a"}`)
	})

	req, _ := testClient.NewRequest("GET", "/", nil)
	res, _ := testClient.Do(req, nil)
	_, err := ioutil.ReadAll(res.Body)

	if err != nil {
		t.Errorf("Error on parsing HTTP Response = %v", err.Error())
	} else if res.StatusCode != 200 {
		t.Errorf("Response code = %v, want %v", res.StatusCode, 200)
	}
}

func TestClient_Do_HTTPError(t *testing.T) {
	setup()
	defer teardown()

	testMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", 400)
	})

	req, _ := testClient.NewRequest("GET", "/", nil)
	_, err := testClient.Do(req, nil)

	if err == nil {
		t.Error("Expected HTTP 400 error.")
	}
}

// Test handling of an error caused by the internal http client's Do() function.
// A redirect loop is pretty unlikely to occur within the Gerrit API, but does allow us to exercise the right code path.
func TestClient_Do_RedirectLoop(t *testing.T) {
	setup()
	defer teardown()

	testMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})

	req, _ := testClient.NewRequest("GET", "/", nil)
	_, err := testClient.Do(req, nil)

	if err == nil {
		t.Error("Expected error to be returned.")
	}
	if err, ok := err.(*url.Error); !ok {
		t.Errorf("Expected a URL error; got %+v.", err)
	}
}

func TestClient_GetBaseURL_WithURL(t *testing.T) {
	u, err := url.Parse(testJIRAInstanceURL)
	if err != nil {
		t.Errorf("URL parsing -> Got an error: %s", err)
	}

	c, err := NewClient(nil, testJIRAInstanceURL)
	if err != nil {
		t.Errorf("Client creation -> Got an error: %s", err)
	}
	if c == nil {
		t.Error("Expected a client. Got none")
	}

	if b := c.GetBaseURL(); !reflect.DeepEqual(b, *u) {
		t.Errorf("Base URLs are not equal. Expected %+v, got %+v", *u, b)
	}
}

// REMOVED : This actually calls a live URL.  It's not a unit test.
// I'm also not really sure what it's testing.
// func TestClient_Do_PagingInfoEmptyByDefault(t *testing.T) {
// 	c, _ := NewClient(nil, testJIRAInstanceURL)
// 	req, _ := c.NewRequest("GET", "/", nil)
// 	t.Errorf("%v\n", req)
// 	type foo struct {
// 		A string
// 	}
// 	body := new(foo)

// 	resp, _ := c.Do(req, body)

// 	if resp.StartAt != 0 {
// 		t.Errorf("StartAt not equal to 0")
// 	}
// 	if resp.MaxResults != 0 {
// 		t.Errorf("StartAt not equal to 0")
// 	}
// 	if resp.Total != 0 {
// 		t.Errorf("StartAt not equal to 0")
// 	}
// }

func TestBasicAuthTransport(t *testing.T) {
	setup()
	defer teardown()

	username, password := "username", "password"

	testMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok {
			t.Errorf("request does not contain basic auth credentials")
		}
		if u != username {
			t.Errorf("request contained basic auth username %q, want %q", u, username)
		}
		if p != password {
			t.Errorf("request contained basic auth password %q, want %q", p, password)
		}
	})

	tp := &BasicAuthTransport{
		Username: username,
		Password: password,
	}

	basicAuthClient, _ := NewClient(tp.Client(), testServer.URL)
	req, _ := basicAuthClient.NewRequest("GET", ".", nil)
	basicAuthClient.Do(req, nil)
}

func TestBasicAuthTransport_transport(t *testing.T) {
	// default transport
	tp := &BasicAuthTransport{}
	if tp.transport() != http.DefaultTransport {
		t.Errorf("Expected http.DefaultTransport to be used.")
	}

	// custom transport
	tp = &BasicAuthTransport{
		Transport: &http.Transport{},
	}
	if tp.transport() == http.DefaultTransport {
		t.Errorf("Expected custom transport to be used.")
	}
}

// Test that the cookie in the transport is the cookie returned in the header
func TestCookieAuthTransport_SessionObject_Exists(t *testing.T) {
	setup()
	defer teardown()

	testCookie := &http.Cookie{Name: "test", Value: "test"}

	testMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()

		if len(cookies) < 1 {
			t.Errorf("No cookies set")
		}

		if cookies[0].Name != testCookie.Name {
			t.Errorf("Cookie names don't match, expected %v, got %v", testCookie.Name, cookies[0].Name)
		}

		if cookies[0].Value != testCookie.Value {
			t.Errorf("Cookie values don't match, expected %v, got %v", testCookie.Value, cookies[0].Value)
		}
	})

	tp := &CookieAuthTransport{
		Username:      "username",
		Password:      "password",
		AuthURL:       "https://some.jira.com/rest/auth/1/session",
		SessionObject: []*http.Cookie{testCookie},
	}

	basicAuthClient, _ := NewClient(tp.Client(), testServer.URL)
	req, _ := basicAuthClient.NewRequest("GET", ".", nil)
	basicAuthClient.Do(req, nil)
}

// Test that if no cookie is in the transport, it checks for a cookie
func TestCookieAuthTransport_SessionObject_DoesNotExist(t *testing.T) {
	setup()
	defer teardown()

	testCookie := &http.Cookie{Name: "does_not_exist", Value: "does_not_exist"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.SetCookie(w, testCookie)
		w.Write([]byte(`OK`))
	}))
	defer ts.Close()

	testMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()

		if len(cookies) < 1 {
			t.Errorf("No cookies set")
		}

		if cookies[0].Name != testCookie.Name {
			t.Errorf("Cookie names don't match, expected %v, got %v", testCookie.Name, cookies[0].Name)
		}

		if cookies[0].Value != testCookie.Value {
			t.Errorf("Cookie values don't match, expected %v, got %v", testCookie.Value, cookies[0].Value)
		}
	})

	tp := &CookieAuthTransport{
		Username: "username",
		Password: "password",
		AuthURL:  ts.URL,
	}

	basicAuthClient, _ := NewClient(tp.Client(), testServer.URL)
	req, _ := basicAuthClient.NewRequest("GET", ".", nil)
	basicAuthClient.Do(req, nil)
}
