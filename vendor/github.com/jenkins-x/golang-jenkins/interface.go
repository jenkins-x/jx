package gojenkins

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

// JenkinsClient is the interface for interacting with jenkins
//go:generate pegomock generate github.com/jenkins-x/golang-jenkins JenkinsClient -o mocks/jenkins_client.go --package gojenkins_test
type JenkinsClient interface {
	GetJobs() ([]Job, error)
	GetJob(string) (Job, error)
	GetJobURLPath(string) string
	IsErrNotFound(error) bool
	BaseURL() string
	SetHTTPClient(*http.Client)
	Post(string, url.Values, interface{}) (err error)
	GetJobConfig(string) (JobItem, error)
	GetBuild(Job, int) (Build, error)
	GetLastBuild(Job) (Build, error)
	StopBuild(Job, int) error
	GetMultiBranchJob(string, string, string) (Job, error)
	GetJobByPath(...string) (Job, error)
	GetOrganizationScanResult(int, Job) (string, error)
	CreateJob(JobItem, string) error
	Reload() error
	Restart() error
	SafeRestart() error
	QuietDown() error
	CreateJobWithXML(string, string) error
	CreateFolderJobWithXML(string, string, string) error
	GetCredential(string) (*Credentials, error)
	CreateCredential(string, string, string) error
	DeleteJob(Job) error
	UpdateJob(JobItem, string) error
	RemoveJob(string) error
	AddJobToView(string, Job) error
	CreateView(ListView) error
	Build(Job, url.Values) error
	GetBuildConsoleOutput(Build) ([]byte, error)
	GetQueue() (Queue, error)
	GetArtifact(Build, Artifact) ([]byte, error)
	SetBuildDescription(Build, string) error
	GetComputerObject() (ComputerObject, error)
	GetComputers() ([]Computer, error)
	GetComputer(string) (Computer, error)
	GetBuildURL(Job, int) string
	GetLogFromURL(string, int64, *LogData) error
	TailLog(string, io.Writer, time.Duration, time.Duration) error
	TailLogFunc(string, io.Writer) ConditionFunc
	NewLogPoller(string, io.Writer) *LogPoller
}
