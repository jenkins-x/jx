package pipline_events

import v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"

type PipelineEventsProvider interface {
	SendActivity(a *v1.PipelineActivity) error
	SendRelease(a *v1.Release) error
}
