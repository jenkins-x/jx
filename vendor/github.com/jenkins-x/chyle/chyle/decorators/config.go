package decorators

// codebeat:disable[TOO_MANY_IVARS]

// Config centralizes config needed for each decorator
type Config struct {
	CUSTOMAPI   customAPIConfig
	GITHUBISSUE githubIssueConfig
	JIRAISSUE   jiraIssueConfig
	ENV         envConfig
	SHELL       shellConfig
}

// Features gives which decorators are enabled
type Features struct {
	ENABLED     bool
	CUSTOMAPI   bool
	JIRAISSUE   bool
	GITHUBISSUE bool
	ENV         bool
	SHELL       bool
}

// codebeat:enable[TOO_MANY_IVARS]
