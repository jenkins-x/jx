package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// ModifyRequirementsFn callback for modifying requirements
type ModifyRequirementsFn func(requirements *helm.Requirements) error

// ConfigureGitFolderFn callback to optionally configure git before its used for creating commits and PRs
type ConfigureGitFolderFn func(dir string, gitInfo *gits.GitRepository, gitAdapter gits.Gitter) error

type CreateEnvPullRequestFn func(env *v1.Environment, modifyRequirementsFn ModifyRequirementsFn, branchNameText string, title string, message string, pullRequestInfo *gits.PullRequestInfo) (*gits.PullRequestInfo, error)

func (o *CommonOptions) createEnvironmentPullRequest(env *v1.Environment, modifyRequirementsFn ModifyRequirementsFn,
	branchNameText *string, title *string, message *string, pullRequestInfo *gits.PullRequestInfo,
	configGitFn ConfigureGitFolderFn) (*gits.PullRequestInfo, error) {
	var answer *gits.PullRequestInfo
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

	branchName := o.Git().ConvertToValidBranchName(asText(branchNameText))
	base := source.Ref
	if base == "" {
		base = "master"
	}

	if exists {
		if configGitFn != nil {
			err = configGitFn(dir, gitInfo, o.Git())
			if err != nil {
				return answer, err
			}
		}
		// lets check the git remote URL is setup correctly
		err = o.Git().SetRemoteURL(dir, "origin", gitURL)
		if err != nil {
			return answer, err
		}
		err = o.Git().Stash(dir)
		if err != nil {
			return answer, err
		}
		err = o.Git().Checkout(dir, base)
		if err != nil {
			return answer, err
		}
		err = o.Git().Pull(dir)
		if err != nil {
			return answer, err
		}
	} else {
		err := os.MkdirAll(dir, DefaultWritePermissions)
		if err != nil {
			return answer, fmt.Errorf("Failed to create directory %s due to %s", dir, err)
		}
		err = o.Git().Clone(gitURL, dir)
		if err != nil {
			return answer, err
		}
		if configGitFn != nil {
			err = configGitFn(dir, gitInfo, o.Git())
			if err != nil {
				return answer, err
			}
		}
		if base != "master" {
			err = o.Git().Checkout(dir, base)
			if err != nil {
				return answer, err
			}
		}

		// TODO lets fork if required???
	}
	branchNames, err := o.Git().RemoteBranchNames(dir, "remotes/origin/")
	if err != nil {
		return answer, fmt.Errorf("Failed to load remote branch names: %s", err)
	}
	//log.Infof("Found remote branch names %s\n", strings.Join(branchNames, ", "))
	if util.StringArrayIndex(branchNames, branchName) >= 0 {
		// lets append a UUID as the branch name already exists
		branchName += "-" + string(uuid.NewUUID())
	}
	err = o.Git().CreateBranch(dir, branchName)
	if err != nil {
		return answer, err
	}
	err = o.Git().Checkout(dir, branchName)
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
	if err != nil {
		return answer, err
	}

	err = helm.SaveRequirementsFile(requirementsFile, requirements)
	if err != nil {
		return answer, err
	}

	err = o.Git().Add(dir, "*", "*/*")
	if err != nil {
		return answer, err
	}
	changed, err := o.Git().HasChanges(dir)
	if err != nil {
		return answer, err
	}
	if !changed {
		log.Warnf("%s\n", "No changes made to the GitOps Environment source code. Code must be up to date!")
		return answer, nil
	}
	err = o.Git().CommitDir(dir, asText(message))
	if err != nil {
		return answer, err
	}
	// lets rebase an existing PR
	if pullRequestInfo != nil {
		remoteBranch := pullRequestInfo.PullRequestArguments.Head
		err = o.Git().ForcePushBranch(dir, branchName, remoteBranch)
		return pullRequestInfo, err
	}

	err = o.Git().Push(dir)
	if err != nil {
		return answer, err
	}

	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return answer, err
	}

	gitKind, err := o.GitServerKind(gitInfo)
	if err != nil {
		return answer, err
	}

	provider, err := gitInfo.PickOrCreateProvider(authConfigSvc, "user name to submit the Pull Request", o.BatchMode, gitKind, o.Git(), o.In, o.Out, o.Err)
	if err != nil {
		return answer, err
	}

	gha := &gits.GitPullRequestArguments{
		GitRepository: gitInfo,
		Title:         asText(title),
		Body:          asText(message),
		Base:          base,
		Head:          branchName,
	}

	pr, err := provider.CreatePullRequest(gha)
	if err != nil {
		return answer, err
	}
	log.Infof("Created Pull Request: %s\n\n", util.ColorInfo(pr.URL))
	return &gits.PullRequestInfo{
		GitProvider:          provider,
		PullRequest:          pr,
		PullRequestArguments: gha,
	}, nil
}

func (o *CommonOptions) registerEnvironmentCRD() error {
	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterEnvironmentCRD(apisClient)
	return err
}

// modifyDevEnvironment performs some mutation on the Development environemnt to modify team settings
func (o *CommonOptions) modifyDevEnvironment(jxClient versioned.Interface, ns string, fn func(env *v1.Environment) error) error {
	env, err := kube.EnsureDevEnvironmentSetup(jxClient, ns)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure that dev environment is setup for namespace '%s'", ns)
	}
	if env == nil {
		return fmt.Errorf("No Development environment found in namespace %s", ns)
	}
	err = fn(env)
	if err != nil {
		return errors.Wrap(err, "failed to call the callback function for dev environment")
	}
	_, err = jxClient.JenkinsV1().Environments(ns).Update(env)
	if err != nil {
		return fmt.Errorf("Failed to update Development environment in namespace %s: %s", ns, err)
	}
	return nil
}

func asText(text *string) string {
	if text != nil {
		return *text
	}
	return ""
}
