package gits

const (
	// KindBitBucketCloud git kind for BitBucket Cloud
	KindBitBucketCloud  = "bitbucketcloud"
	// KindBitBucketServer git kind for BitBucket Server
	KindBitBucketServer = "bitbucketserver"
	// KindGitea git kind for gitea
	KindGitea           = "gitea"
	// KindGitlab git kind for gitlab
	KindGitlab          = "gitlab"
	// KindGitHub git kind for github
	KindGitHub          = "github"
	// KindGitFake git kind for fake git
	KindGitFake         = "fakegit"
	// KindUnknown git kind for unknown git
	KindUnknown         = "unknown"

	// BitbucketCloudURL the default URL for BitBucket Cloud
	BitbucketCloudURL = "https://bitbucket.org"

	// FakeGitURL the default URL for the fake git provider
	FakeGitURL = "https://fake.git"
)

var (
	KindGits = []string{KindBitBucketCloud, KindBitBucketServer, KindGitea, KindGitHub, KindGitlab}
)
