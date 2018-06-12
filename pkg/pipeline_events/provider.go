package pipline_events

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
)

type PipelineEventsProvider interface {
	SendActivity(a *v1.PipelineActivity) error
	SendRelease(a *v1.Release) error
}
