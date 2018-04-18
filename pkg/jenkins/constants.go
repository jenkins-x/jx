package jenkins

const (
	DefaultJenkinsCredentialsPrefix = "jenkins-x-"

	DefaultJenkinsfile = "Jenkinsfile"

	Chartmuseum = "chartmuseum"

	// BranchPatternMasterPRsAndFeatures only match master, PRs and features
	BranchPatternMasterPRsAndFeatures = "master|PR-.*|feature.*"

	// BranchPatternMatchEverything matches everything
	BranchPatternMatchEverything = ".*"
)
