package releases

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/gits"
)

var (
	dependencyUpdateRegex = regexp.MustCompile(`^(?m:chore\(dependencies\): update (.*) from ([\w\.]*) to ([\w\.]*)$)`)
	slugLinkRegex         = regexp.MustCompile(`^(?:([\w-]*?)?\/?([\w-]+)|(https?):\/\/([\w\.]*)\/([\w-]*)\/([\w-]*))$`)
	//slugLinkRegex = regexp.MustCompile(``)
	slugRegex = regexp.MustCompile(`^(\w*)?\/(\w*)$`)
)

// ReleaseDownloadCount returns the total number of downloads for the given set of releases
func ReleaseDownloadCount(releases []*gits.GitRelease) int {
	count := 0
	for _, release := range releases {
		count += release.DownloadCount
	}
	return count
}

//DependencyUpdate describes an dependency update message from the commit log
type DependencyUpdate struct {
	URL         string
	Owner       string
	Host        string
	Repo        string
	FromVersion string
	ToVersion   string
}

func (d *DependencyUpdate) String() string {
	return fmt.Sprintf("%s/%s %s", d.Owner, d.Host, d.ToVersion)
}

// ParseDependencyUpdateMessage parses commit messages, and if it's a dependency update message parses it
//
// A complete update message looks like:
//
// chore(dependencies): update ((<owner>/)?<repo>|https://<gitHost>/<owner>/<repo>) from <fromVersion> to <toVersion>
//
// <description of update method>
//
// <fromVersion>, <toVersion> and <repo> are required fields. The markdown URL format is optional, and a plain <owner>/<repo>
// can be used.
func ParseDependencyUpdateMessage(msg string) (*DependencyUpdate, error) {
	matches := dependencyUpdateRegex.FindStringSubmatch(msg)
	if matches == nil {
		// string does not match at all
		return nil, nil
	}
	if len(matches) != 4 {
		return nil, errors.Errorf("parsing %s as dependency update message", msg)
	}
	slug := matches[1]
	var urlScheme, urlHost, owner, repo string
	slugMatches := slugLinkRegex.FindStringSubmatch(slug)
	if len(slugMatches) == 7 {
		if slugMatches[6] != "" {
			owner = slugMatches[5]
			repo = slugMatches[6]
			urlScheme = slugMatches[3]
			urlHost = slugMatches[4]
		} else {
			owner = slugMatches[1]
			repo = slugMatches[2]
		}

	} else {
		return nil, nil
	}

	return &DependencyUpdate{
		Owner:       owner,
		Repo:        repo,
		Host:        urlHost,
		URL:         fmt.Sprintf("%s://%s/%s/%s", urlScheme, urlHost, owner, repo),
		FromVersion: matches[2],
		ToVersion:   matches[3],
	}, nil
}
