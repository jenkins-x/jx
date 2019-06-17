package codecov

// Variables in this file are set by the build and used by the instrumented binary for upload to codecov
var (
	// Flag is the flag to be passed to codecov
	Flag string
	// Slug is the slug to be passed to codecov
	Slug string
	// Branch is the branch on which the changeset is built to be passed to codecov
	Branch string
	// Sha is the sha of the last commit of the changeset to be passed to codecov
	Sha string
	// BuildNumber is the number of the build of the changeset to be passed to codecov
	BuildNumber string
	// PullRequestNumber is the number of the pull request for the changeset (or empty) to be passed to codecov
	PullRequestNumber string
	// Tag is the tag name of the changeset (or empty) to be passed to codecov
	Tag string
)
