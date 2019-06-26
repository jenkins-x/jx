package jenkins

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	gojenkins "github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

func GetJenkinsClient(server auth.Server) (gojenkins.JenkinsClient, error) {
	user, err := server.GetCurrentUser()
	if err != nil {
		return nil, errors.Wrap(err, "Getting the current user")
	}
	auth := &gojenkins.Auth{
		Username:    user.Username,
		ApiToken:    user.ApiToken,
		BearerToken: user.BearerToken,
	}
	jenkins := gojenkins.NewJenkins(auth, server.URL)

	// handle insecure TLS for minishift
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	jenkins.SetHTTPClient(httpClient)
	return jenkins, nil
}

func PrintGetTokenFromURL(out io.Writer, tokenUrl string) (int, error) {
	return fmt.Fprintf(out, "Please go to %s and click %s to get your API Token\n", util.ColorInfo(tokenUrl), util.ColorInfo("Show API Token"))
}

func JenkinsTokenURL(url string) string {
	tokenUrl := util.UrlJoin(url, "/me/configure")
	return tokenUrl
}

func JenkinsApiURL(url string) string {
	return util.UrlJoin(url, "/api")
}

// JenkinsLoginURL returns the Jenkins login URL
func JenkinsLoginURL(url string) string {
	return util.UrlJoin(url, "/login")
}

// IsMultiBranchProject returns true if this job is a multi branch project
func IsMultiBranchProject(job *gojenkins.Job) bool {
	return job != nil && job.Class == "org.jenkinsci.plugins.workflow.multibranch.WorkflowMultiBranchProject"
}

// LoadAllJenkinsJobs Loads all the jobs in full from the Jenkins client
func LoadAllJenkinsJobs(jenkinsClient gojenkins.JenkinsClient) ([]*gojenkins.Job, error) {
	answer := []*gojenkins.Job{}
	jobs, err := jenkinsClient.GetJobs()
	if err != nil {
		return answer, err
	}

	for _, j := range jobs {
		childJobs, err := loadChildJobs(jenkinsClient, j.Name)
		if err != nil {
			return answer, err
		}
		answer = append(answer, childJobs...)
	}
	return answer, nil
}

func loadChildJobs(jenkinsClient gojenkins.JenkinsClient, name string) ([]*gojenkins.Job, error) {
	answer := []*gojenkins.Job{}
	job, err := jenkinsClient.GetJob(name)
	if err != nil {
		return answer, err
	}
	answer = append(answer, &job)

	if job.Jobs != nil {
		for _, child := range job.Jobs {
			childJobs, err := loadChildJobs(jenkinsClient, job.FullName+"/"+child.Name)
			if err != nil {
				return answer, err
			}
			answer = append(answer, childJobs...)
		}
	}
	return answer, nil
}

// JobName returns the Jenkins job name starting with the given prefix
func JobName(prefix string, j *gojenkins.Job) string {
	name := j.FullName
	if name == "" {
		name = j.Name
	}
	if prefix != "" {
		name = prefix + "/" + name
	}
	return name
}

// IsPipeline checks if the job is a pipeline job
func IsPipeline(j *gojenkins.Job) bool {
	return strings.Contains(j.Class, "Job")
}

// SwitchJenkinsBaseURL sometimes a Jenkins server does not know its external URL so lets switch the base URL of the job
// URL to use the known working baseURL of the jenkins server
func SwitchJenkinsBaseURL(jobURL string, baseURL string) string {
	if jobURL == "" {
		return baseURL
	}
	if baseURL == "" {
		return jobURL
	}
	u, err := url.Parse(jobURL)
	if err != nil {
		log.Logger().Warnf("failed to parse Jenkins Job URL %s due to: %s", jobURL, err)
		return jobURL
	}

	u2, err := url.Parse(baseURL)
	if err != nil {
		log.Logger().Warnf("failed to parse Jenkins base URL %s due to: %s", baseURL, err)
		return jobURL
	}
	u.Host = u2.Host
	u.Scheme = u2.Scheme
	return u.String()
}
