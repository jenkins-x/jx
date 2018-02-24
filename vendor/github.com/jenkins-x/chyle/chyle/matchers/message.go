package matchers

import (
	"regexp"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// message is commit message matcher
type message struct {
	regexp *regexp.Regexp
}

func (m message) Match(commit *object.Commit) bool {
	return m.regexp.MatchString(commit.Message)
}

func newMessage(re *regexp.Regexp) Matcher {
	return message{re}
}

// removePGPKey fix library issue that don't trim PGP key from message
func removePGPKey(message string) string {
	return regexp.MustCompile("(?s).*?-----END PGP SIGNATURE-----\n\n").ReplaceAllString(message, "")
}
