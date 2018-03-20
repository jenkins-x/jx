package gits

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
	"net/url"
	"strings"
)

const (
	GitHubHost = "github.com"

	gitPrefix = "git@"
)

type GitRepositoryInfo struct {
	URL          string
	Scheme       string
	Host         string
	Organisation string
	Name         string
}

func (i *GitRepositoryInfo) IsGitHub() bool {
	return GitHubHost == i.Host
}

// PullRequestURL returns the URL of a pull request of the given name/number
func (i *GitRepositoryInfo) PullRequestURL(prName string) string {
	return util.UrlJoin("https://"+i.Host, i.Organisation, i.Name, "pull", prName)
}

// HttpCloneURL returns the HTTPS git URL this repository
func (i *GitRepositoryInfo) HttpCloneURL() string {
	return i.HttpsURL() + ".git"
}

// HttpURL returns the URL to browse this repository in a web browser
func (i *GitRepositoryInfo) HttpURL() string {
	host := i.Host
	if !strings.Contains(host, ":/") {
		host = "http://" + host
	}
	return util.UrlJoin(host, i.Organisation, i.Name)
}

// HttpsURL returns the URL to browse this repository in a web browser
func (i *GitRepositoryInfo) HttpsURL() string {
	host := i.Host
	if !strings.Contains(host, ":/") {
		host = "https://" + host
	}
	return util.UrlJoin(host, i.Organisation, i.Name)
}

// HostURL returns the URL to the host
func (i *GitRepositoryInfo) HostURL() string {
	answer := i.Host
	if !strings.Contains(answer, ":/") {
		// lets find the scheme from the URL
		u := i.URL
		if u != "" {
			u2, err := url.Parse(u)
			if err != nil {
				fmt.Printf("WARNING: could not parse URL: %s due to %s\n", u, err)
			}
			s := u2.Scheme
			if s != "" {
				if !strings.HasSuffix(s, "://") {
					s += "://"
				}
				return s + answer
			}
		}
		return "https://" + answer
	}
	return answer
}

func (i *GitRepositoryInfo) HostURLWithoutUser() string {
	if i.Host == "github.com" {
		return i.Host
	}
	u := i.URL
	if u != "" {
		u2, err := url.Parse(u)
		if err == nil {
			u2.User = nil
			u2.Path = ""
			return u2.String()
		}
	}
	return i.HttpsURL()
}

// PipelinePath returns the pipeline path for the master branch which can be used to query
// pipeline logs in `jx get build logs myPipelinePath`
func (i *GitRepositoryInfo) PipelinePath() string {
	return i.Organisation + "/" + i.Name + "/master"
}

// ParseGitURL attempts to parse the given text as a URL or git URL-like string to determine
// the protocol, host, organisation and name
func ParseGitURL(text string) (*GitRepositoryInfo, error) {
	answer := GitRepositoryInfo{
		URL: text,
	}
	u, err := url.Parse(text)
	if err == nil && u != nil {
		answer.Host = u.Host

		// lets default to github
		if answer.Host == "" {
			answer.Host = GitHubHost
		}
		if answer.Scheme == "" {
			answer.Scheme = "https"
		}
		answer.Scheme = u.Scheme
		return parsePath(u.Path, &answer)
	}

	// handle git@ kinds of URIs
	if strings.HasPrefix(text, gitPrefix) {
		t := strings.TrimPrefix(text, gitPrefix)
		t = strings.TrimPrefix(t, "/")
		t = strings.TrimPrefix(t, "/")
		t = strings.TrimSuffix(t, "/")
		t = strings.TrimSuffix(t, ".git")

		arr := util.RegexpSplit(t, ":|/")
		if len(arr) >= 3 {
			answer.Scheme = "git"
			answer.Host = arr[0]
			answer.Organisation = arr[1]
			answer.Name = arr[2]
			return &answer, nil
		}
	}
	return nil, fmt.Errorf("Could not parse git url %s", text)
}

func parsePath(path string, info *GitRepositoryInfo) (*GitRepositoryInfo, error) {
	arr := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(arr) >= 2 {
		info.Organisation = arr[0]
		info.Name = strings.TrimSuffix(arr[1], ".git")
		return info, nil
	} else {
		return info, fmt.Errorf("Invalid path %s could not determine organisation and repository name", path)
	}
}
