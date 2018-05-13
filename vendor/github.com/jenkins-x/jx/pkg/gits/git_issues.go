package gits

import (
	"strconv"
	"strings"
	"time"
)

// IsClosedSince returns true if the issue has been closed since the given da
func (i *GitIssue) IsClosedSince(t time.Time) bool {
	t2 := i.ClosedAt
	state := i.State
	if t2 != nil && state != nil {
		lowerState := strings.ToLower(*state)
		return (t2.Equal(t) || t2.After(t)) && strings.HasPrefix(lowerState, "clos")
	}
	return false
}

// Name returns the textual name of the issue
func (i *GitIssue) Name() string {
	if i.Key != "" {
		return i.Key
	}
	n := i.Number
	if n != nil {
		return "#" + strconv.Itoa(*n)
	}
	return "N/A"
}

// FilterIssuesClosedSince returns a filtered slice of all the issues closed since the given date
func FilterIssuesClosedSince(issues []*GitIssue, t time.Time) []*GitIssue {
	answer := []*GitIssue{}
	for _, issue := range issues {
		if issue != nil && issue.IsClosedSince(t) {
			answer = append(answer, issue)
		}
	}
	return answer
}
