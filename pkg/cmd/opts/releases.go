package opts

import (
	"fmt"
	"regexp"

	"github.com/jenkins-x/jx/pkg/gits/releases"

	"github.com/jenkins-x/jx/pkg/gits"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/pkg/errors"
)

var (
	dependencyUpdateRegex = regexp.MustCompile(`^(?m:chore\(dependencies\): update (.*) from ([\w\.]*) to ([\w\.]*)$)`)
	slugLinkRegex         = regexp.MustCompile(`^(?:([\w-]*?)?\/?([\w-]+)|(https?):\/\/([\w\.]*)\/([\w-]*)\/([\w-]*))(?::([\w-]*))?$`)
	//slugLinkRegex = regexp.MustCompile(``)
	slugRegex = regexp.MustCompile(`^(\w*)?\/(\w*)$`)
)

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
func (o *CommonOptions) ParseDependencyUpdateMessage(msg string, commitURL string) (*v1.DependencyUpdate, error) {
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
	if len(slugMatches) == 8 {
		commitInfo, err := gits.ParseGitURL(commitURL)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing %s", commitURL)
		}
		if slugMatches[6] != "" {
			owner = slugMatches[5]
			repo = slugMatches[6]
			urlScheme = slugMatches[3]
			urlHost = slugMatches[4]
		} else {
			owner = slugMatches[1]
			repo = slugMatches[2]
		}
		if owner == "" {
			owner = commitInfo.Organisation
		}
		if urlScheme == "" {
			urlScheme = commitInfo.Scheme
		}
		if urlHost == "" {
			urlHost = commitInfo.Host
		}
		update := &v1.DependencyUpdate{
			Owner:       owner,
			Repo:        repo,
			Host:        urlHost,
			URL:         fmt.Sprintf("%s://%s/%s/%s", urlScheme, urlHost, owner, repo),
			FromVersion: matches[2],
			ToVersion:   matches[3],
			Component:   slugMatches[7],
		}
		provider, _, err := o.CreateGitProviderForURLWithoutKind(update.URL)
		if err != nil {
			return nil, errors.Wrapf(err, "creating git provider for %s", update.URL)
		}

		toRelease, err := releases.GetRelease(update.ToVersion, update.Owner, update.Repo, provider)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if toRelease != nil {
			update.ToReleaseHTMLURL = toRelease.HTMLURL
			update.ToReleaseName = toRelease.Name
		}
		fromRelease, err := releases.GetRelease(update.FromVersion, update.Owner, update.Repo, provider)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if fromRelease != nil {
			update.FromReleaseHTMLURL = fromRelease.HTMLURL
			update.FromReleaseName = fromRelease.Name
		}
		return update, nil
	}
	return nil, nil
}
