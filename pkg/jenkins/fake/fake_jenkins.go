package fake

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jenkins-x/golang-jenkins"
)

// NewFakeJenkins contains the state of the fake JenkinsClient
type FakeJenkins struct {
	baseURL string
	Jobs    []gojenkins.Job
	JobMap  map[string]*gojenkins.Job
}

// NewFakeJenkins creates a fake JenkinsClient that can be used in tests
func NewFakeJenkins() *FakeJenkins {
	return &FakeJenkins{
		JobMap: map[string]*gojenkins.Job{},
	}
}

func (j *FakeJenkins) GetJobs() ([]gojenkins.Job, error) {
	return j.Jobs, nil
}

func (j *FakeJenkins) GetJob(name string) (gojenkins.Job, error) {
	for _, job := range j.Jobs {
		if job.Name == name {
			return job, nil
		}
	}
	return gojenkins.Job{}, j.notFoundf("job: %s", name)
}

func (j *FakeJenkins) CreateJobWithXML(jobXml string, folder string) error {
	folderJob := j.getOrCreateFolderJob(folder)
	if folderJob != nil {
		return nil
	}
	return j.notFoundf("job: %s", folder)
}

func (j *FakeJenkins) CreateFolderJobWithXML(jobXml string, folder string, jobName string) error {
	folderJob := j.getOrCreateFolderJob(folder)

	for _, job := range folderJob.Jobs {
		if job.Name == jobName {
			return nil
		}
	}
	folderJob.Jobs = append(folderJob.Jobs, gojenkins.Job{
		Name: jobName,
		Url:  folderJob.Url + "/job/" + folder,
	})
	return nil
}

func (j *FakeJenkins) getOrCreateFolderJob(folder string) *gojenkins.Job {
	var folderJob *gojenkins.Job
	for i, job := range j.Jobs {
		if job.Name == folder {
			folderJob = &j.Jobs[i]
			break
		}
	}
	if folderJob == nil {
		j.Jobs = append(j.Jobs, gojenkins.Job{
			Name: folder,
			Url:  "/job/" + folder,
		})
		folderJob = &j.Jobs[len(j.Jobs)-1]
	}
	return folderJob
}

func (j *FakeJenkins) GetJobURLPath(name string) string {
	return gojenkins.FullPath(name)
}

func (j *FakeJenkins) IsErrNotFound(err error) bool {
	te, ok := err.(gojenkins.APIError)
	return ok && te.StatusCode == 404
}

func (j *FakeJenkins) BaseURL() string {
	return j.baseURL
}

func (j *FakeJenkins) SetHTTPClient(*http.Client) {
}

func (j *FakeJenkins) Post(string, url.Values, interface{}) (err error) {
	return nil
}

func (j *FakeJenkins) GetJobConfig(name string) (gojenkins.JobItem, error) {
	return gojenkins.JobItem{}, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) GetBuild(gojenkins.Job, int) (gojenkins.Build, error) {
	return gojenkins.Build{}, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) GetLastBuild(gojenkins.Job) (gojenkins.Build, error) {
	return gojenkins.Build{}, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) StopBuild(gojenkins.Job, int) error {
	return nil
}

func (j *FakeJenkins) GetMultiBranchJob(string, string, string) (gojenkins.Job, error) {
	return gojenkins.Job{}, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) GetJobByPath(names ...string) (gojenkins.Job, error) {
	jobs := j.Jobs
	lastIdx := len(names) - 1
	for idx, name := range names {
		found := false
		for _, job := range jobs {
			if job.Name == name {
				if idx >= lastIdx {
					return job, nil
				}
				jobs = job.Jobs
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return gojenkins.Job{}, j.notFoundf("job for path %s", strings.Join(names, "/"))
}

func (j *FakeJenkins) GetOrganizationScanResult(int, gojenkins.Job) (string, error) {
	return "", fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) CreateJob(gojenkins.JobItem, string) error {
	return nil
}

func (j *FakeJenkins) Reload() error {
	return nil
}

func (j *FakeJenkins) Restart() error {
	return nil
}

func (j *FakeJenkins) SafeRestart() error {
	return nil
}

func (j *FakeJenkins) QuietDown() error {
	return nil
}

func (j *FakeJenkins) GetCredential(string) (*gojenkins.Credentials, error) {
	return nil, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) CreateCredential(string, string, string) error {
	return nil
}

func (j *FakeJenkins) DeleteJob(gojenkins.Job) error {
	return nil
}

func (j *FakeJenkins) UpdateJob(gojenkins.JobItem, string) error {
	return nil
}

func (j *FakeJenkins) RemoveJob(string) error {
	return nil
}

func (j *FakeJenkins) AddJobToView(string, gojenkins.Job) error {
	return nil
}

func (j *FakeJenkins) CreateView(gojenkins.ListView) error {
	return nil
}

func (j *FakeJenkins) Build(gojenkins.Job, url.Values) error {
	return nil
}

func (j *FakeJenkins) GetBuildConsoleOutput(gojenkins.Build) ([]byte, error) {
	return nil, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) GetQueue() (gojenkins.Queue, error) {
	return gojenkins.Queue{}, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) GetArtifact(gojenkins.Build, gojenkins.Artifact) ([]byte, error) {
	return nil, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) SetBuildDescription(gojenkins.Build, string) error {
	return nil
}

func (j *FakeJenkins) GetComputerObject() (gojenkins.ComputerObject, error) {
	return gojenkins.ComputerObject{}, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) GetComputers() ([]gojenkins.Computer, error) {
	return nil, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) GetComputer(string) (gojenkins.Computer, error) {
	return gojenkins.Computer{}, fmt.Errorf("Not implemented!")
}

func (j *FakeJenkins) GetBuildURL(gojenkins.Job, int) string {
	return ""
}

func (j *FakeJenkins) GetLogFromURL(string, int64, *gojenkins.LogData) error {
	return nil
}

func (j *FakeJenkins) TailLog(string, io.Writer, time.Duration, time.Duration) error {
	return nil
}

func (j *FakeJenkins) TailLogFunc(string, io.Writer) gojenkins.ConditionFunc {
	return nil
}

func (j *FakeJenkins) NewLogPoller(string, io.Writer) *gojenkins.LogPoller {
	return nil
}

func (j *FakeJenkins) notFound(message string) error {
	return fmt.Errorf("not found: %s", message)
}

func (j *FakeJenkins) notFoundf(message string, arguments ...interface{}) error {
	return j.notFound(fmt.Sprintf(message, arguments...))
}
