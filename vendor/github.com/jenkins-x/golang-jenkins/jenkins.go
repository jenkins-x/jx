package gojenkins

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Auth struct {
	Username    string
	ApiToken    string
	BearerToken string
}

type Jenkins struct {
	auth    *Auth
	baseUrl string
	client  *http.Client
}

type Credential struct {
	Credentials Credentials `json:"credentials"`
}

type Credentials struct {
	Scope    string `json:"scope"`
	Id       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Class    string `json:"$class"`
}

type APIError struct {
	Status     string
	StatusCode int
}

func (err APIError) Error() string {
	return err.Status
}

func (jenkins *Jenkins) IsErrNotFound(err error) bool {
	te, ok := err.(APIError)
	return ok && te.StatusCode == 404
}

func NewJenkins(auth *Auth, baseUrl string) *Jenkins {

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	return &Jenkins{
		auth:    auth,
		baseUrl: baseUrl,
		client:  client,
	}
}

// BaseURL returns the base Jenkins URL
func (jenkins *Jenkins) BaseURL() string {
	return jenkins.baseUrl
}

// SetHTTPClient with timeouts or insecure transport, etc.
func (jenkins *Jenkins) SetHTTPClient(client *http.Client) {
	jenkins.client = client
}

func (jenkins *Jenkins) buildUrl(path string, params url.Values) (requestUrl string) {
	requestUrl = jenkins.baseUrl + path + "/api/json"
	if params != nil {
		queryString := params.Encode()
		if queryString != "" {
			requestUrl = requestUrl + "?" + queryString
		}
	}

	return
}

func (jenkins *Jenkins) buildUrlNoBase(path string, params url.Values) (requestUrl string) {
	requestUrl = path + "/api/json"
	if params != nil {
		queryString := params.Encode()
		if queryString != "" {
			requestUrl = requestUrl + "?" + queryString
		}
	}

	return
}

func (jenkins *Jenkins) buildUrlRaw(path string, params url.Values) (requestUrl string) {
	requestUrl = jenkins.baseUrl + path
	if params != nil {
		queryString := params.Encode()
		if queryString != "" {
			requestUrl = requestUrl + "?" + queryString
		}
	}

	return
}

// checkCrumb - checks if `useCrumb` is enabled and if so, retrieves crumb field and value and updates request header
func (jenkins *Jenkins) checkCrumb(req *http.Request) (*http.Request, error) {

	// api - store jenkins api useCrumbs response
	api := struct {
		UseCrumbs bool `json:"useCrumbs"`
	}{}

	err := jenkins.get("/api/json", url.Values{"tree": []string{"useCrumbs"}}, &api)
	if err != nil {
		return req, err
	}

	if !api.UseCrumbs {
		// CSRF Protection is not enabled
		return req, nil
	}

	// get crumb field and value
	crumb := struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}{}

	err = jenkins.get("/crumbIssuer", nil, &crumb)
	if err != nil {
		return req, err
	}

	// update header
	req.Header.Set(crumb.CrumbRequestField, crumb.Crumb)

	return req, nil
}

func (jenkins *Jenkins) sendRequest(req *http.Request) (*http.Response, error) {
	if jenkins.auth != nil {
		if jenkins.auth.BearerToken != "" {
			req.Header.Add("Authorization", "Bearer "+jenkins.auth.BearerToken)
		} else {
			req.SetBasicAuth(jenkins.auth.Username, jenkins.auth.ApiToken)
		}
	}
	return jenkins.client.Do(req)
}

func (jenkins *Jenkins) parseXmlResponse(resp *http.Response, body interface{}) (err error) {
	defer resp.Body.Close()

	if resp.StatusCode != 302 && resp.StatusCode != 405 && resp.StatusCode > 201 {
		return APIError{resp.Status, resp.StatusCode}
	}

	if body == nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return xml.Unmarshal(data, body)
}

func (jenkins *Jenkins) parseXmlResponseWithWrapperElement(resp *http.Response, body interface{}, rootElementName string) (err error) {
	defer resp.Body.Close()

	if resp.StatusCode != 302 && resp.StatusCode != 405 && resp.StatusCode > 201 {
		return APIError{resp.Status, resp.StatusCode}
	}

	if body == nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	xmlText := strings.TrimSpace(string(data))
	if strings.Index(xmlText, "<?") == 0 {
		idx := strings.Index(xmlText, "?>")
		if idx < 0 {
			return fmt.Errorf("Could not find matching '?>' characters to strip the XML pragma!")
		}
		xmlText = xmlText[idx+2:]
	}
	//log.Printf("Parsing XML: %s", xmlText)
	wrappedDoc := "<" + rootElementName + ">\n" + xmlText + "\n</" + rootElementName + ">"
	return xml.Unmarshal([]byte(wrappedDoc), body)
}

func (jenkins *Jenkins) parseResponse(resp *http.Response, body interface{}) (err error) {
	defer resp.Body.Close()

	if resp.StatusCode != 302 && resp.StatusCode != 405 && resp.StatusCode > 201 {
		return APIError{resp.Status, resp.StatusCode}
	}
	if body == nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	return json.Unmarshal(data, body)
}

func (jenkins *Jenkins) parseResponseRaw(resp *http.Response, body interface{}) (dataStr string, err error) {
	defer resp.Body.Close()

	if resp.StatusCode != 302 && resp.StatusCode != 405 && resp.StatusCode > 201 {
		return
	}

	if body == nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	dataStr = string(data[:])
	return
}

func (jenkins *Jenkins) get(path string, params url.Values, body interface{}) (err error) {
	requestUrl := jenkins.buildUrl(path, params)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return
	}
	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	return jenkins.parseResponse(resp, body)
}

func (jenkins *Jenkins) getUrl(path string, params url.Values, body interface{}) (err error) {
	requestUrl := jenkins.buildUrlNoBase(path, params)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return
	}
	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	return jenkins.parseResponse(resp, body)
}

// uses the actual URL passed in rather than appending api/json
func (jenkins *Jenkins) getRaw(path string, params url.Values, body interface{}) (data string, err error) {
	requestUrl := jenkins.buildUrlRaw(path, params)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return
	}

	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	data, err = jenkins.parseResponseRaw(resp, body)

	return
}

func (jenkins *Jenkins) getXml(path string, params url.Values, body interface{}) (err error) {
	requestUrl := jenkins.buildUrl(path, params)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return
	}

	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	return jenkins.parseXmlResponse(resp, body)
}

func (jenkins *Jenkins) getConfigXml(path string, params url.Values, body interface{}) (err error) {
	requestUrl := jenkins.buildUrl(path, params)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return
	}

	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	// lets add a dummy item wrapper xml to make the xml parsing easier
	return jenkins.parseXmlResponseWithWrapperElement(resp, body, "item")
}

func (jenkins *Jenkins) post(path string, params url.Values, body interface{}) (err error) {
	requestUrl := jenkins.buildUrl(path, params)

	req, err := http.NewRequest("POST", requestUrl, nil)
	if err != nil {
		return
	}

	if _, err := jenkins.checkCrumb(req); err != nil {
		return err
	}

	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	return jenkins.parseResponse(resp, body)
}

func (jenkins *Jenkins) postUrl(path string, params url.Values, body interface{}) (err error) {
	requestUrl := jenkins.buildUrlNoBase(path, params)

	req, err := http.NewRequest("POST", requestUrl, nil)
	if err != nil {
		return
	}

	if _, err := jenkins.checkCrumb(req); err != nil {
		return err
	}

	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	return jenkins.parseResponse(resp, body)
}

func (jenkins *Jenkins) postXml(path string, params url.Values, xmlBody io.Reader, body interface{}) (err error) {
	requestUrl := jenkins.baseUrl + path
	if params != nil {
		queryString := params.Encode()
		if queryString != "" {
			requestUrl = requestUrl + "?" + queryString
		}
	}

	req, err := http.NewRequest("POST", requestUrl, xmlBody)
	if err != nil {
		return
	}

	if _, err := jenkins.checkCrumb(req); err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/xml")
	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	return jenkins.parseXmlResponse(resp, body)
}

func (jenkins *Jenkins) postJson(path string, data url.Values, body interface{}) (err error) {
	requestUrl := jenkins.baseUrl + path

	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}

	if _, err := jenkins.checkCrumb(req); err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := jenkins.sendRequest(req)
	if err != nil {
		return
	}
	return jenkins.parseResponse(resp, body)
}

// GetJobs returns all jobs you can read.
func (jenkins *Jenkins) GetJobs() ([]Job, error) {
	var payload = struct {
		Jobs []Job `json:"jobs"`
	}{}
	err := jenkins.get("", nil, &payload)
	return payload.Jobs, err
}

// GetJob returns a job which has specified name.
func (jenkins *Jenkins) GetJob(name string) (job Job, err error) {
	err = jenkins.get(FullPath(name), nil, &job)
	return
}

// GetJobUrl returns the URL path for the job with the specified name.
func (jenkins *Jenkins) GetJobURLPath(name string) string {
	return FullPath(name)
}

//GetJobConfig returns a maven job, has the one used to create Maven job
func (jenkins *Jenkins) GetJobConfig(name string) (job JobItem, err error) {
	err = jenkins.getConfigXml(FullPath(name)+"/config.xml", nil, &job)
	return
}

// GetBuild returns a number-th build result of specified job.
func (jenkins *Jenkins) GetBuild(job Job, number int) (build Build, err error) {
	err = jenkins.getUrl(fmt.Sprintf("%s%d", job.Url, number), nil, &build)
	return
}

// GetLastBuild returns the last build of specified job.
func (jenkins *Jenkins) GetLastBuild(job Job) (build Build, err error) {
	err = jenkins.getUrl(fmt.Sprintf("%slastBuild", job.Url), nil, &build)
	return
}

// Stops the given build.
func (jenkins *Jenkins) StopBuild(job Job, number int) error {
	return jenkins.postUrl(fmt.Sprintf("%s%d/stop", job.Url, number), nil, nil)
}


func (jenkins *Jenkins) GetMultiBranchJob(organisationJobName, multibranchJobName, branch string) (job Job, err error) {
	err = jenkins.get(fmt.Sprintf("/job/%s/job/%s/job/%s", organisationJobName, multibranchJobName, branch), nil, &job)
	return
}

// GetJobByPath looks up the jenkins job via the one or more paths
func (jenkins *Jenkins) GetJobByPath(path ...string) (job Job, err error) {
	fullPath := FullJobPath(path...)
	err = jenkins.get(fullPath, nil, &job)
	return
}

// FullJobPath returns the full job path URL for the given paths
func FullJobPath(path ...string) string {
	buffer := bytes.NewBufferString("")
	for _, p := range path {
		buffer.WriteString("/job/")
		t := strings.TrimPrefix(p, "/")
		buffer.WriteString(strings.TrimSuffix(t, "/"))
	}
	return buffer.String()
}

// FullJobPath returns the full job path URL for the given paths
func FullPath(job string) string {
	paths := strings.Split(job, "/")
	return FullJobPath(paths...)
}

// GetLastBuild returns the last build of specified job.
func (jenkins *Jenkins) GetOrganizationScanResult(retries int, job Job) (status string, err error) {

	// wait util the scan has finished - not found anything on the jenkins remote API to do this but must be a better way?
	err = RetryAfter(retries, func() error {

		data, err := jenkins.getRaw(fmt.Sprintf("/job/%s/computation/consoleText", job.Name), nil, &job)
		if err != nil {
			return err
		}
		dataArray := strings.Split(data, "\n")
		lastLine := string(dataArray[len(dataArray)-2])
		if strings.Contains(lastLine, "Finished") {
			return nil
		}
		return errors.New("Scan not finished yet")
	}, time.Second*5)

	data, err := jenkins.getRaw(fmt.Sprintf("/job/%s/computation/consoleText", job.Name), nil, &job)
	if err != nil {
		return "", err
	}
	dataArray := strings.Split(data, "\n")
	lastLine := string(dataArray[len(dataArray)-2])

	return strings.Replace(lastLine, "Finished: ", "", 1), nil
}

// Create a new job
func (jenkins *Jenkins) CreateJob(jobItem JobItem, jobName string) error {
	jobItemXml, err := JobToXml(jobItem)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(jobItemXml)
	params := url.Values{"name": []string{jobName}}

	return jenkins.postXml("/createItem", params, reader, nil)
}

// Reload will reload the configuration from disk
func (jenkins *Jenkins) Reload() error {
	return jenkins.postPath( "/reload")
}

// Restart
func (jenkins *Jenkins) Restart() error {
	return jenkins.postPath( "/restart")
}

// SafeRestart waits for the jenkins server to be quiet then restarts it
func (jenkins *Jenkins) SafeRestart() error {
	return jenkins.postPath( "/safeRestart")
}

// QuietDown
func (jenkins *Jenkins) QuietDown() error {
	return jenkins.postPath( "/quietDown")
}

func (jenkins *Jenkins) postPath(path string) error {
	reader := bytes.NewReader([]byte{})
	params := url.Values{}
	return jenkins.postXml(path, params, reader, nil)
}

// Create a new job
func (jenkins *Jenkins) CreateJobWithXML(jobItemXml string, jobName string) error {

	reader := bytes.NewReader([]byte(jobItemXml))
	params := url.Values{"name": []string{jobName}}

	return jenkins.postXml("/createItem", params, reader, nil)
}

// Create a new job in a folder
func (jenkins *Jenkins) CreateFolderJobWithXML(jobItemXml string, folder string, jobName string) error {

	reader := bytes.NewReader([]byte(jobItemXml))
	params := url.Values{"name": []string{jobName}}

	return jenkins.postXml("/job/"+folder+"/createItem", params, reader, nil)
}

// Get a credentials
func (jenkins *Jenkins) GetCredential(id string) (c *Credentials, err error) {
	c = &Credentials{}
	requestUrl := "/credentials/store/system/domain/_/credentials/" + strings.TrimPrefix(id, "/")
	err = jenkins.get(requestUrl, nil, c)
	return
}

// Create a new credentials
func (jenkins *Jenkins) CreateCredential(id, username, pass string) error {
	c := Credentials{
		Scope:    "GLOBAL",
		Id:       id,
		Username: username,
		Password: pass,
		Class:    "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl",
	}

	b, err := json.Marshal(Credential{Credentials: c})
	if err != nil {
		return err
	}
	data := url.Values{}
	data.Set("json", string(b))

	return jenkins.postJson("/credentials/store/system/domain/_/createCredentials", data, nil)
}

// Delete a job
func (jenkins *Jenkins) DeleteJob(job Job) error {
	return jenkins.postUrl(fmt.Sprintf("%sdoDelete", job.Url), nil, nil)
}

// Update a job
func (jenkins *Jenkins) UpdateJob(jobItem JobItem, jobName string) error {
	jobItemXml, err := JobToXml(jobItem)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(jobItemXml)
	params := url.Values{"name": []string{}}

	return jenkins.postXml(jenkins.GetJobURLPath(jobName), params, reader, nil)
}

// Remove a job
func (jenkins *Jenkins) RemoveJob(jobName string) error {
	reader := bytes.NewReader([]byte{})
	params := url.Values{}
	url := jenkins.GetJobURLPath(jobName) + "/doDelete"
	return jenkins.postXml(url, params, reader, nil)
}

// Add job to view
func (jenkins *Jenkins) AddJobToView(viewName string, job Job) error {
	params := url.Values{"name": []string{job.Name}}
	return jenkins.post(fmt.Sprintf("/view/%s/addJobToView", viewName), params, nil)
}

// Create a new view
func (jenkins *Jenkins) CreateView(listView ListView) error {
	xmlListView, _ := xml.Marshal(listView)
	reader := bytes.NewReader(xmlListView)
	params := url.Values{"name": []string{listView.Name}}

	return jenkins.postXml("/createView", params, reader, nil)
}

// Create a new build for this job.
// Params can be nil.
func (jenkins *Jenkins) Build(job Job, params url.Values) error {

	if hasParams(job) {
		return jenkins.postUrl(fmt.Sprintf("%sbuildWithParameters", job.Url), params, nil)
	} else {
		return jenkins.postUrl(fmt.Sprintf("%sbuild", job.Url), params, nil)
	}
}

// Get the console output from a build.
func (jenkins *Jenkins) GetBuildConsoleOutput(build Build) ([]byte, error) {
	requestUrl := fmt.Sprintf("%s/consoleText", build.Url)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	res, err := jenkins.sendRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// GetQueue returns the current build queue from Jenkins
func (jenkins *Jenkins) GetQueue() (queue Queue, err error) {
	err = jenkins.get(fmt.Sprintf("/queue"), nil, &queue)
	return
}

// GetArtifact return the content of a build artifact
func (jenkins *Jenkins) GetArtifact(build Build, artifact Artifact) ([]byte, error) {
	requestUrl := fmt.Sprintf("%s/artifact/%s", build.Url, artifact.RelativePath)
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	res, err := jenkins.sendRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// SetBuildDescription sets the description of a build
func (jenkins *Jenkins) SetBuildDescription(build Build, description string) error {
	requestUrl := fmt.Sprintf("%ssubmitDescription?description=%s", build.Url, url.QueryEscape(description))
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return err
	}

	res, err := jenkins.sendRequest(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("Unexpected response: expected '200' but received '%d'", res.StatusCode)
	}

	return nil
}

// GetComputerObject returns the main ComputerObject
func (jenkins *Jenkins) GetComputerObject() (co ComputerObject, err error) {
	err = jenkins.get(fmt.Sprintf("/computer"), nil, &co)
	return
}

// GetComputers returns the list of all Computer objects
func (jenkins *Jenkins) GetComputers() ([]Computer, error) {
	var payload = struct {
		Computers []Computer `json:"computer"`
	}{}
	err := jenkins.get("/computer", nil, &payload)
	return payload.Computers, err
}

// GetComputer returns a Computer object with a specified name.
func (jenkins *Jenkins) GetComputer(name string) (computer Computer, err error) {
	err = jenkins.get(fmt.Sprintf("/computer/%s", name), nil, &computer)
	return
}

// hasParams returns a boolean value indicating if the job is parameterized
func hasParams(job Job) bool {
	for _, action := range job.Actions {
		if len(action.ParameterDefinitions) > 0 {
			return true
		}
	}
	return false
}
