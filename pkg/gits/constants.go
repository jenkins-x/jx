package gits

const (
	KindBitBucketCloud  = "bitbucketcloud"
	KindBitBucketServer = "bitbucketserver"
	KindGitea           = "gitea"
	KindGitlab          = "gitlab"
	KindGitHub          = "github"

	BitbucketCloudURL = "https://bitbucket.org"
)

var (
	KindGits = []string{KindBitBucketCloud, KindBitBucketServer, KindGitea, KindGitHub, KindGitlab}
)
