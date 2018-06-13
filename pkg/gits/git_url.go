package gits

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
)

const (
	GitHubHost = "github.com"
	GitHubURL  = "https://github.com"

	gitPrefix = "git@"
)

type GitRepositoryInfo struct {
	URL          string
	Scheme       string
	Host         string
	Organisation string
	Name         string
	Project      string
}

func (i *GitRepositoryInfo) IsGitHub() bool {
	return GitHubHost == i.Host || strings.HasSuffix(i.URL, "https://github.com")
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
				// probably a git@ URL
				return "https://" + answer
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
	u := i.URL
	if u != "" {
		u2, err := url.Parse(u)
		if err == nil {
			u2.User = nil
			u2.Path = ""
			return u2.String()
		}

	}
	host := i.Host
	if !strings.Contains(host, ":/") {
		host = "https://" + host
	}
	return host
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
	trimPath := strings.TrimSuffix(path, "/")
	trimPath = strings.TrimSuffix(trimPath, ".git")
	arr := strings.Split(trimPath, "/")
	arrayLength := len(arr)
	if arrayLength >= 2 {
		info.Organisation = arr[arrayLength-2]
		info.Project = arr[arrayLength-2]
		info.Name = arr[arrayLength-1]

		return info, nil
	}

	return info, fmt.Errorf("Invalid path %s could not determine organisation and repository name", path)
}

// SaasGitKind returns the kind for SaaS git providers or "" if the URL could not be deduced
func SaasGitKind(gitServiceUrl string) string {
	switch gitServiceUrl {
	case "http://github.com":
		return KindGitHub
	case "https://github.com":
		return KindGitHub
	case "http://bitbucket.org":
		return KindBitBucketCloud
	case BitbucketCloudURL:
		return KindBitBucketCloud
	default:
		return ""
	}
}
