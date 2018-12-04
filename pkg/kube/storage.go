package kube

import "strings"

const (
	// ClassificationLogs stores build logs
	ClassificationLogs = "logs"

	// ClassificationTest stores test results/rports
	ClassificationTests= "tests"

	// ClassificationCoverage stores code coverage results/reports
	ClassificationCoverage = "coverage"
)

var (
	// Classifications the common classification names
	Classifications = []string{
		ClassificationCoverage, ClassificationTests, ClassificationLogs,
	}

	// ClassificationValues the classification values as a string
	ClassificationValues = strings.Join(Classifications, ", ")

)
