package issues

const (
	Bugzilla = "bugzilla"
	Jira     = "jira"
	Trello   = "trello"
	Git      = "git"
)

var (
	IssueTrackerKinds = []string{Bugzilla, Jira, Trello}
)
