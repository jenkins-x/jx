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
		return info, fmt.Errorf("Invalid path %s could not determine organisation and repository name")
	}
}
