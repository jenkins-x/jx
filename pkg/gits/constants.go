package gits

const (
	KindBitBucketCloud  = "bitbucketcloud"
	KindBitBucketServer = "bitbucketserver"
	KindGitea           = "gitea"
	KindGitlab          = "gitlab"
	KindGitHub          = "github"

	DateFormat = "January 2 2006"

	BitbucketCloudURL = "https://bitbucket.org"
)

var (
	KindGits = []string{KindBitBucketCloud, KindBitBucketServer, KindGitea, KindGitHub, KindGitlab}
)
