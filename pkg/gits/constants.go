package gits

const (
	KindBitBucket       = "bitbucket"
	KindBitBucketServer = "bitbucketserver"
	KindGitea           = "gitea"
	KindGitlab          = "gitlab"
	KindGitHub          = "github"

	DateFormat = "January 2 2006"

	BitbucketCloudURL = "https://bitbucket.org"
)

var (
	KindGits = []string{KindBitBucket, KindGitea, KindGitHub, KindGitlab}
)
