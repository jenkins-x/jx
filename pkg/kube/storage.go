package kube

import "strings"

const (
	// ClassificationLogs stores build logs
	ClassificationLogs = "logs"

	// ClassificationTest stores test results/reports
	ClassificationTests = "tests"

	// ClassificationCoverage stores code coverage results/reports
	ClassificationCoverage = "coverage"

	// ClassificationReports stores code coverage results/reports
	ClassificationReports = "reports"
)

var (
	// Classifications the common classification names
	Classifications = []string{
		ClassificationCoverage, ClassificationTests, ClassificationLogs, ClassificationReports,
	}

	// ClassificationValues the classification values as a string
	ClassificationValues = strings.Join(Classifications, ", ")
)
