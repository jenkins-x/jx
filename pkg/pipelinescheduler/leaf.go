package pipelinescheduler

import jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

type SchedulerLeaf struct {
	*jenkinsv1.SchedulerSpec
	Org  string
	Repo string
}
