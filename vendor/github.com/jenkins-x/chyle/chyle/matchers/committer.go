package matchers

import (
	"regexp"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// committer is commit committer matcher
type committer struct {
	regexp *regexp.Regexp
}

func (c committer) Match(commit *object.Commit) bool {
	return c.regexp.MatchString(commit.Committer.String())
}

func newCommitter(re *regexp.Regexp) Matcher {
	return committer{re}
}
