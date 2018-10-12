package governance

import (
	"strconv"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
)

const complianceCheckContext = "compliance-check"

func NotifyComplianceState(commitRef jenkinsv1.ComplianceCheckCommitReference, state string, targetUrl string, description string, comment string, gitProvider gits.GitProvider, gitRepoInfo *gits.GitRepositoryInfo) (status *gits.GitRepoStatus, err error) {

	if err != nil {
		return &gits.GitRepoStatus{}, err
	}

	status = &gits.GitRepoStatus{
		Description: description,
		State:       state,
		TargetURL:   targetUrl,
		Context:     complianceCheckContext,
	}

	oldStatuses, err := gitProvider.ListCommitStatus(gitRepoInfo.Organisation, gitRepoInfo.Name, commitRef.SHA)
	if err != nil {
		return &gits.GitRepoStatus{}, err
	}
	for _, o := range oldStatuses {
		if o.Context == complianceCheckContext {
			status.ID = o.ID
		}
	}
	log.Infof("Status %s for compliance check for pull request %s (%s) on %s\n", state, commitRef.PullRequest, commitRef.SHA, commitRef.GitURL)
	_, err = gitProvider.UpdateCommitStatus(gitRepoInfo.Organisation, gitRepoInfo.Name, commitRef.SHA, status)
	if err != nil {
		return &gits.GitRepoStatus{}, err
	}
	if comment != "" {
		prn, err := strconv.Atoi(commitRef.PullRequest)
		if err != nil {
			return &gits.GitRepoStatus{}, err
		}
		pr, err := gitProvider.GetPullRequest(gitRepoInfo.Organisation, gitRepoInfo, prn)
		if err != nil {
			return &gits.GitRepoStatus{}, err
		}
		err = gitProvider.AddPRComment(pr, comment)
		if err != nil {
			return &gits.GitRepoStatus{}, err
		}
	}
	return status, nil
}
