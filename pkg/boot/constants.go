package boot

const (
	// PullRequestLabel is the label used on pull requests created by boot
	PullRequestLabel = "jx/boot"
	// OverrideTLSWarningEnvVarName is an environment variable set in BDD tests to override the error (in batch mode)
	// that is created if TLS is not enabled
	OverrideTLSWarningEnvVarName = "TESTING_ONLY_OVERRIDE_TLS_WARNING"
)
