package versionstreamrepo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/gits"
	gitconfig "gopkg.in/src-d/go-git.v4/config"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// CloneJXVersionsRepo clones the jenkins-x versions repo to a local working dir
func CloneJXVersionsRepo(versionRepository string, versionRef string, settings *v1.TeamSettings, gitter gits.Gitter, batchMode bool, advancedMode bool, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, string, error) {
	dir, versionRef, err := cloneJXVersionsRepo(versionRepository, versionRef, settings, gitter, batchMode, advancedMode, in, out, errOut)
	if err != nil {
		return "", "", errors.Wrapf(err, "")
	}
	if versionRef != "" {
		resolved, err := resolveRefToTag(dir, versionRef, gitter)
		if err != nil {
			return "", "", errors.WithStack(err)
		}
		return dir, resolved, nil
	}
	return dir, "", nil
}

func cloneJXVersionsRepo(versionRepository string, versionRef string, settings *v1.TeamSettings, gitter gits.Gitter, batchMode bool, advancedMode bool, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, string, error) {
	surveyOpts := survey.WithStdio(in, out, errOut)
	configDir, err := util.ConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("error determining config dir %v", err)
	}
	wrkDir := filepath.Join(configDir, "jenkins-x-versions")

	if settings != nil {
		if versionRepository == "" {
			versionRepository = settings.VersionStreamURL
		}
		if versionRef == "" {
			versionRef = settings.VersionStreamRef
		}
	}
	if versionRepository == "" {
		versionRepository = config.DefaultVersionsURL
	}

	log.Logger().Debugf("Current configuration dir: %s", configDir)
	log.Logger().Debugf("versionRepository: %s git ref: %s", versionRepository, versionRef)

	// If the repo already exists let's try to fetch the latest version
	if exists, err := util.DirExists(wrkDir); err == nil && exists {
		repo, err := git.PlainOpen(wrkDir)
		if err != nil {
			log.Logger().Errorf("Error opening %s", wrkDir)
			_, err := deleteAndReClone(wrkDir, versionRepository, versionRef, gitter, out)
			if err != nil {
				return "", "", errors.WithStack(err)
			}
		}
		remote, err := repo.Remote("origin")
		if err != nil {
			log.Logger().Errorf("Error getting remote origin")
			dir, err := deleteAndReClone(wrkDir, versionRepository, versionRef, gitter, out)
			if err != nil {
				return "", "", errors.WithStack(err)
			}
			return dir, versionRef, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		remoteRefs := "+refs/heads/master:refs/remotes/origin/master"
		if versionRef != "" {
			remoteRefs = "+refs/heads/" + versionRef + ":refs/remotes/origin/" + versionRef
		}
		err = remote.FetchContext(ctx, &git.FetchOptions{
			RefSpecs: []gitconfig.RefSpec{
				gitconfig.RefSpec(remoteRefs),
			},
		})

		// The repository is up to date
		if err == git.NoErrAlreadyUpToDate {
			if versionRef != "" {
				err = gitter.Checkout(wrkDir, versionRef)
				if err != nil {
					dir, err := deleteAndReClone(wrkDir, versionRepository, versionRef, gitter, out)
					if err != nil {
						return "", "", errors.WithStack(err)
					}
					return dir, versionRef, nil
				}
			}
			return wrkDir, versionRef, nil
		} else if err != nil {
			dir, err := deleteAndReClone(wrkDir, versionRepository, versionRef, gitter, out)
			if err != nil {
				return "", "", errors.WithStack(err)
			}
			return dir, versionRef, nil
		}

		pullLatest := false
		if batchMode {
			pullLatest = true
		} else if advancedMode {
			confirm := &survey.Confirm{
				Message: "A local Jenkins X versions repository already exists, pull the latest?",
				Default: true,
			}
			err = survey.AskOne(confirm, &pullLatest, nil, surveyOpts)
			if err != nil {
				log.Logger().Errorf("Error confirming if we should pull latest, skipping %s", wrkDir)
			}
		} else {
			pullLatest = true
			log.Logger().Infof(util.QuestionAnswer("A local Jenkins X versions repository already exists, pulling the latest", util.YesNo(pullLatest)))
		}
		if pullLatest {
			w, err := repo.Worktree()
			if err == nil {
				err := w.Pull(&git.PullOptions{RemoteName: "origin"})
				if err != nil {
					return "", "", errors.Wrap(err, "pulling the latest")
				}
			}
		}
		if versionRef != "" {
			err = gitter.Checkout(wrkDir, versionRef)
			if err != nil {
				dir, err := deleteAndReClone(wrkDir, versionRepository, versionRef, gitter, out)
				if err != nil {
					return "", "", errors.WithStack(err)
				}
				return dir, versionRef, nil
			}
		}
		return wrkDir, versionRef, err
	}
	dir, err := deleteAndReClone(wrkDir, versionRepository, versionRef, gitter, out)
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	return dir, versionRef, nil
}

func deleteAndReClone(wrkDir string, versionRepository string, referenceName string, gitter gits.Gitter, fw terminal.FileWriter) (string, error) {
	log.Logger().Info("Deleting and cloning the Jenkins X versions repo")
	err := os.RemoveAll(wrkDir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to delete dir %s: %s\n", wrkDir, err.Error())
	}
	err = os.MkdirAll(wrkDir, util.DefaultWritePermissions)
	if err != nil {
		return "", errors.Wrapf(err, "failed to ensure directory is created %s", wrkDir)
	}
	_, err = clone(wrkDir, versionRepository, referenceName, gitter, fw)
	if err != nil {
		return "", err
	}
	return wrkDir, err
}

func clone(wrkDir string, versionRepository string, referenceName string, gitter gits.Gitter, fw terminal.FileWriter) (string, error) {
	if referenceName == "" || referenceName == "master" {
		referenceName = "refs/heads/master"
	} else if !strings.Contains(referenceName, "/") {
		if strings.HasPrefix(referenceName, "PR-") {
			prNumber := strings.TrimPrefix(referenceName, "PR-")

			log.Logger().Infof("Cloning the Jenkins X versions repo %s with PR: %s to %s", util.ColorInfo(versionRepository), util.ColorInfo(referenceName), util.ColorInfo(wrkDir))
			return "", shallowCloneGitRepositoryToDir(wrkDir, versionRepository, prNumber, "", gitter)
		}
		log.Logger().Infof("Cloning the Jenkins X versions repo %s with revision %s to %s", util.ColorInfo(versionRepository), util.ColorInfo(referenceName), util.ColorInfo(wrkDir))

		err := gitter.Clone(versionRepository, wrkDir)
		if err != nil {
			return "", errors.Wrapf(err, "failed to clone repository: %s to dir %s", versionRepository, wrkDir)
		}
		cmd := util.Command{
			Dir:  wrkDir,
			Name: "git",
			Args: []string{"fetch", "origin", referenceName},
		}
		_, err = cmd.RunWithoutRetry()
		if err != nil {
			return "", errors.Wrapf(err, "failed to git fetch origin %s for repo: %s in dir %s", referenceName, versionRepository, wrkDir)
		}
		err = gitter.Checkout(wrkDir, "FETCH_HEAD")
		if err != nil {
			return "", errors.Wrapf(err, "failed to checkout FETCH_HEAD of repo: %s in dir %s", versionRepository, wrkDir)
		}
		return "", nil
	}
	log.Logger().Infof("Cloning the Jenkins X versions repo %s with ref %s to %s", util.ColorInfo(versionRepository), util.ColorInfo(referenceName), util.ColorInfo(wrkDir))
	_, err := git.PlainClone(wrkDir, false, &git.CloneOptions{
		URL:           versionRepository,
		ReferenceName: plumbing.ReferenceName(referenceName),
		SingleBranch:  true,
		Progress:      nil,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to clone reference: %s", referenceName)
	}
	return "", err
}

func shallowCloneGitRepositoryToDir(dir string, gitURL string, pullRequestNumber string, revision string, gitter gits.Gitter) error {
	if pullRequestNumber != "" {
		log.Logger().Infof("shallow cloning pull request %s of repository %s to temp dir %s", gitURL,
			pullRequestNumber, dir)
		err := gitter.ShallowClone(dir, gitURL, "", pullRequestNumber)
		if err != nil {
			return errors.Wrapf(err, "shallow cloning pull request %s of repository %s to temp dir %s\n", gitURL,
				pullRequestNumber, dir)
		}
	} else if revision != "" {
		log.Logger().Infof("shallow cloning revision %s of repository %s to temp dir %s", gitURL,
			revision, dir)
		err := gitter.ShallowClone(dir, gitURL, revision, "")
		if err != nil {
			return errors.Wrapf(err, "shallow cloning revision %s of repository %s to temp dir %s\n", gitURL,
				revision, dir)
		}
	} else {
		log.Logger().Infof("shallow cloning master of repository %s to temp dir %s", gitURL, dir)
		err := gitter.ShallowClone(dir, gitURL, "", "")
		if err != nil {
			return errors.Wrapf(err, "shallow cloning master of repository %s to temp dir %s\n", gitURL, dir)
		}
	}

	return nil
}

func resolveRefToTag(dir string, commitish string, gitter gits.Gitter) (string, error) {
	err := gitter.FetchTags(dir)
	if err != nil {
		return "", errors.Wrapf(err, "fetching tags")
	}
	resolved, _, err := gitter.Describe(dir, true, commitish, "0")
	if err != nil {
		return "", errors.Wrapf(err, "running git describe %s --abbrev=0", commitish)
	}
	if resolved != "" {
		return resolved, nil
	}
	return resolved, nil
}
