package matchers

import (
	"regexp"
)

// codebeat:disable[TOO_MANY_IVARS]

// Config centralizes config needed for each matcher
type Config struct {
	MESSAGE   *regexp.Regexp
	COMMITTER *regexp.Regexp
	AUTHOR    *regexp.Regexp
	TYPE      string
}

// Features gives which matchers are enabled
type Features struct {
	ENABLED   bool
	MESSAGE   bool
	COMMITTER bool
	AUTHOR    bool
	TYPE      bool
}

// codebeat:enable[TOO_MANY_IVARS]
