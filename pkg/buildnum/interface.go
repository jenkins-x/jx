package buildnum

import "github.com/jenkins-x/jx/pkg/kube"

// A BuildNumberIssuer generates build numbers for activities.
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/buildnum BuildNumberIssuer -o mocks/buildnum.go --generate-matchers
type BuildNumberIssuer interface {
	NextBuildNumber(pipeline kube.PipelineID) (string, error)
}
