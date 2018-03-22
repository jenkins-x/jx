package issues

const (
	Bugzilla = "bugzilla"
	Jira     = "jira"
	Trello   = "trello"
)

var (
	IssueTrackerKinds = []string{Bugzilla, Jira, Trello}
)
