package gits

import (
	"strings"
)

type CommitInfo struct {
	Kind    string
	Feature string
	Message string
}

var (
	// ConventionalCommitTitles textual descriptions for
	// Conventional Commit types: https://conventionalcommits.org/
	ConventionalCommitTitles = map[string]string{
		"feat":     "Features",
		"fix":      "Bug Fixes",
		"perf":     "Performance Improvements",
		"revert":   "Reverts",
		"docs":     "Documentation",
		"style":    "Styles",
		"refactor": "Code Refactoring",
		"test":     "Tests",
		"chore":    "Chores",
	}
)

// ConventionalCommitTypeToTitle returns the title of the conventional commit type
// see: https://conventionalcommits.org/
func ConventionalCommitTypeToTitle(kind string) string {
	answer := ConventionalCommitTitles[strings.ToLower(kind)]
	if answer == "" {
		answer = strings.Title(kind)
	}
	return answer
}


// ParseCommit parses a conventional commit
// see: https://conventionalcommits.org/
func ParseCommit(message string) *CommitInfo {
	answer := &CommitInfo{
		Message: message,
	}

	idx := strings.Index(message, ":")
	if idx > 0 {
		answer.Kind = message[0:idx]

		rest := message[idx + 1:]
		if strings.HasPrefix(rest, "(") {
			idx = strings.Index(rest, ")")
			if idx > 0 {
				answer.Feature = rest[1:idx]
				rest = strings.TrimSpace(rest[idx + 1:])
			}
		}
		answer.Message = rest
	}
	return answer
}

func (c *CommitInfo) Title() string {
	return ConventionalCommitTypeToTitle(c.Kind)
}