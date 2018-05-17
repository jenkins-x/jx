package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// callback for modifying requirements
type ModifyRequirementsFn func(requirements *helm.Requirements) error

func (o *CommonOptions) createEnvironmentPullRequest(env *v1.Environment, modifyRequirementsFn ModifyRequirementsFn, branchNameText string, title string, message string) (*ReleasePullRequestInfo, error) {
	var answer *ReleasePullRequestInfo
	source := &env.Spec.Source
	gitURL := source.URL
	if gitURL == "" {
		return answer, fmt.Errorf("No source git URL")
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return answer, err
	}

	environmentsDir, err := util.EnvironmentsDir()
	if err != nil {
		return answer, err
	}
	dir := filepath.Join(environmentsDir, gitInfo.Organisation, gitInfo.Name)

	// now lets clone the fork and push it...
	exists, err := util.FileExists(dir)
	if err != nil {
		return answer, err
	}

	branchName := gits.ConvertToValidBranchName(branchNameText)
	base := source.Ref
	if base == "" {
		base = "master"
	}

	if exists {
		// lets check the git remote URL is setup correctly
		err = gits.SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return answer, err
		}
		err = gits.GitCmd(dir, "stash")
		if err != nil {
			return answer, err
		}
		err = gits.GitCmd(dir, "checkout", base)
		if err != nil {
			return answer, err
		}
		err = gits.GitCmd(dir, "pull")
		if err != nil {
			return answer, err
		}
	} else {
		err := os.MkdirAll(dir, DefaultWritePermissions)
		if err != nil {
			return answer, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		err = gits.GitClone(gitURL, dir)
		if err != nil {
			return answer, err
		}
		if base != "master" {
			err = gits.GitCmd(dir, "checkout", base)
			if err != nil {
				return answer, err
			}
		}

		// TODO lets fork if required???
		/*
			pushGitURL, err := gits.GitCreatePushURL(gitURL, details.User)
			if err != nil {
				return answer, err
			}
			err = gits.GitCmd(dir, "remote", "add", "upstream", forkEnvGitURL)
			if err != nil {
				return answer, err
			}
			err = gits.GitCmd(dir, "remote", "add", "origin", pushGitURL)
			if err != nil {
				return answer, err
			}
			err = gits.GitCmd(dir, "push", "-u", "origin", "master")
			if err != nil {
				return answer, err
			}
		*/
	}
	branchNames, err := gits.GitGetRemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return answer, fmt.Errorf("Failed to load remote branch names: %s", err)
	}
	o.Printf("Found remote branch names %s\n", strings.Join(branchNames, ", "))
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchName += "-" + string(uuid.NewUUID())
	}
	err = gits.GitCmd(dir, "branch", branchName)
	if err != nil {
		return answer, err
	}
	err = gits.GitCmd(dir, "checkout", branchName)
	if err != nil {
		return answer, err
	}

	requirementsFile, err := helm.FindRequirementsFileName(dir)
	if err != nil {
		return answer, err
	}
	requirements, err := helm.LoadRequirementsFile(requirementsFile)
	if err != nil {
		return answer, err
	}

	err = modifyRequirementsFn(requirements)

	err = helm.SaveRequirementsFile(requirementsFile, requirements)

	err = gits.GitCmd(dir, "add", "*", "*/*")
	if err != nil {
		return answer, err
	}
	changed, err := gits.HasChanges(dir)
	if err != nil {
		return answer, err
	}
	if !changed {
		o.Printf("%s\n", util.ColorWarning("No changes made to the GitOps Environment source code. Code must be up to date!"))
		return answer, nil
	}
	err = gits.GitCommitDir(dir, message)
	if err != nil {
		return answer, err
	}
	err = gits.GitPush(dir)
	if err != nil {
		return answer, err
	}

	authConfigSvc, err := o.Factory.CreateGitAuthConfigService()
	if err != nil {
		return answer, err
	}

	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return answer, err
	}

	provider, err := gitInfo.PickOrCreateProvider(authConfigSvc, "user name to submit the Pull Request", o.BatchMode, gitKind)
	if err != nil {
		return answer, err
	}

	gha := &gits.GitPullRequestArguments{
		Owner: gitInfo.Organisation,
		Repo:  gitInfo.Name,
		Title: title,
		Body:  message,
		Base:  base,
		Head:  branchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
		return answer, err
	}
	o.Printf("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return &ReleasePullRequestInfo{
		GitProvider:          provider,
		PullRequest:          pr,
		PullRequestArguments: gha,
	}, nil
}
