package kube

import (
	"time"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func StartPromote(p *v1.PromoteActivityStep) error {
	if p.StartedTimestamp == nil {
		p.StartedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	if p.Status == v1.ActivityStatusTypeNone {
		p.Status = v1.ActivityStatusTypeRunning
	}
	return nil
}

func CompletePromote(p *v1.PromoteActivityStep) error {
	err := StartPromote(p)
	if err != nil {
		return err
	}
	if p.CompletedTimestamp == nil {
		p.CompletedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	p.Status = v1.ActivityStatusTypeSucceeded
	return nil
}

func FailedPromote(p *v1.PromoteActivityStep) error {
	err := StartPromote(p)
	if err != nil {
		return err
	}
	if p.CompletedTimestamp == nil {
		p.CompletedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	p.Status = v1.ActivityStatusTypeFailed
	return nil
}

func StartPromotionPullRequest(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
	err := StartPromote(ps)
	if err != nil {
		return err
	}
	if p.StartedTimestamp == nil {
		p.StartedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	if a.Spec.WorkflowStatus != v1.ActivityStatusTypeRunning {
		a.Spec.WorkflowStatus = v1.ActivityStatusTypeRunning
	}
	if a.Spec.Status != v1.ActivityStatusTypeRunning {
		a.Spec.Status = v1.ActivityStatusTypeRunning
	}
	if p.Status != v1.ActivityStatusTypeRunning {
		p.Status = v1.ActivityStatusTypeRunning
	}
	return nil
}

func StartPromotionUpdate(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromoteUpdateStep) error {
	err := StartPromote(ps)
	if err != nil {
		return err
	}
	pullRequest := ps.PullRequest
	if pullRequest != nil {
		err = CompletePromotionPullRequest(a, s, ps, pullRequest)
		if err != nil {
			return errors.Wrap(err, "unable to complete promotion pull request")
		}
	}
	if p.StartedTimestamp == nil {
		p.StartedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	if a.Spec.WorkflowStatus != v1.ActivityStatusTypeRunning {
		a.Spec.WorkflowStatus = v1.ActivityStatusTypeRunning
	}
	if a.Spec.Status != v1.ActivityStatusTypeRunning {
		a.Spec.Status = v1.ActivityStatusTypeRunning
	}
	if p.Status != v1.ActivityStatusTypeRunning {
		p.Status = v1.ActivityStatusTypeRunning
	}
	return nil
}

func CompletePromotionPullRequest(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
	if p.StartedTimestamp == nil {
		p.StartedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	if p.CompletedTimestamp == nil {
		p.CompletedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	p.Status = v1.ActivityStatusTypeSucceeded
	return nil
}

func FailedPromotionPullRequest(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromotePullRequestStep) error {
	if p.StartedTimestamp == nil {
		p.StartedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	if p.CompletedTimestamp == nil {
		p.CompletedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	p.Status = v1.ActivityStatusTypeFailed
	return nil
}

func CompletePromotionUpdate(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromoteUpdateStep) error {
	err := CompletePromote(ps)
	if err != nil {
		return err
	}
	pullRequest := ps.PullRequest
	if pullRequest != nil {
		err = CompletePromotionPullRequest(a, s, ps, pullRequest)
		if err != nil {
			return err
		}
	}
	if p.StartedTimestamp == nil {
		p.StartedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	if p.CompletedTimestamp == nil {
		p.CompletedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	p.Status = v1.ActivityStatusTypeSucceeded
	return nil
}

func FailedPromotionUpdate(a *v1.PipelineActivity, s *v1.PipelineActivityStep, ps *v1.PromoteActivityStep, p *v1.PromoteUpdateStep) error {
	err := FailedPromote(ps)
	if err != nil {
		return err
	}
	pullRequest := ps.PullRequest
	if pullRequest != nil {
		err = CompletePromotionPullRequest(a, s, ps, pullRequest)
		if err != nil {
			return err
		}
	}
	if p.StartedTimestamp == nil {
		p.StartedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	if p.CompletedTimestamp == nil {
		p.CompletedTimestamp = &metav1.Time{
			Time: time.Now(),
		}
	}
	p.Status = v1.ActivityStatusTypeFailed
	return nil
}
