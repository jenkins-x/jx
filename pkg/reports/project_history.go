package reports

type ProjectHistory struct {
	LastReport string          `json:"lastReport,omitempty"`
	Reports    []ProjectReport `json:"reports,omitempty"`
}

type ProjectReport struct {
	DateCreated       string `json:"dateCreated,omitempty"`
	DownloadCount     int    `json:"downloadCount,omitempty"`
	IssueCount        int    `json:"issueCount,omitempty"`
	PullRequestCount  int    `json:"pullRequestCount,omitempty"`
	CommitCount       int    `json:"commitCount,omitempty"`
	NewCommitterCount int    `json:"newCommitterCount,omitempty"`
}
