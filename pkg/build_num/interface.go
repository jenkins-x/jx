package build_num

import "github.com/jenkins-x/jx/pkg/kube"

// A BuildNumberIssuer generates build numbers for activities.
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/build_num BuildNumberIssuer -o mocks/build_num.go --generate-matchers
type BuildNumberIssuer interface {
	NextBuildNumber(pipeline kube.PipelineID) (string, error)
}
