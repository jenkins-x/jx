package collector

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// GitCollector stores the state for the git collector
type GitCollector struct {
	gitInfo   *gits.GitRepository
	gitter    gits.Gitter
	gitBranch string
}

// NewGitCollector creates a new git based collector
func NewGitCollector(gitter gits.Gitter, gitURL string, gitBranch string) (Collector, error) {
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return nil, err
	}

	return &GitCollector{
		gitter:    gitter,
		gitInfo:   gitInfo,
		gitBranch: gitBranch,
	}, nil
}

// CollectFiles collects files and returns the URLs
func (c *GitCollector) CollectFiles(patterns []string, outputPath string, basedir string) ([]string, error) {
	urls := []string{}

	gitClient := c.gitter
	storageGitInfo := c.gitInfo
	storageOrg := storageGitInfo.Organisation
	storageRepoName := storageGitInfo.Name

	ghPagesDir, err := cloneGitHubPagesBranchToTempDir(c.gitInfo.URL, gitClient, c.gitBranch)
	if err != nil {
		return urls, err
	}

	repoDir := filepath.Join(ghPagesDir, outputPath)
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return urls, err
	}

	for _, p := range patterns {
		names, err := filepath.Glob(p)
		if err != nil {
			return urls, errors.Wrapf(err, "failed to evaluate glob pattern '%s'", p)
		}
		for _, name := range names {
			toName := name
			if basedir != "" {
				toName, err = filepath.Rel(basedir, name)
				if err != nil {
					return urls, errors.Wrapf(err, "failed to remove basedir %s from %s", basedir, name)
				}
			}
			toFile := filepath.Join(repoDir, toName)
			toDir, _ := filepath.Split(toFile)
			err = os.MkdirAll(toDir, util.DefaultWritePermissions)
			if err != nil {
				return urls, errors.Wrapf(err, "failed to create directory file %s", toDir)
			}
			err = util.CopyFileOrDir(name, toFile, true)
			if err != nil {
				return urls, errors.Wrapf(err, "failed to copy file %s to %s", name, toFile)
			}
		}
	}

	err = filepath.Walk(repoDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				rPath := strings.TrimPrefix(strings.TrimPrefix(path, ghPagesDir), "/")

				if rPath != "" {
					url := c.generateURL(storageOrg, storageRepoName, rPath)
					urls = append(urls, url)
				}
			}
			return nil
		})
	if err != nil {
		return urls, err
	}

	err = gitClient.Add(ghPagesDir, repoDir)
	if err != nil {
		return urls, err
	}
	changes, err := gitClient.HasChanges(ghPagesDir)
	if err != nil {
		return urls, err
	}
	if !changes {
		return urls, nil
	}
	err = gitClient.CommitDir(ghPagesDir, fmt.Sprintf("Publishing files for path %s", outputPath))
	if err != nil {
		fmt.Println(err)
		return urls, err
	}
	err = gitClient.Push(ghPagesDir)
	return urls, err
}

// CollectData collects the data storing it at the given output path and returning the URL
// to access it
func (c *GitCollector) CollectData(data []byte, outputPath string) (string, error) {
	u := ""
	gitClient := c.gitter
	storageGitInfo := c.gitInfo
	storageOrg := storageGitInfo.Organisation
	storageRepoName := storageGitInfo.Name

	ghPagesDir, err := cloneGitHubPagesBranchToTempDir(c.gitInfo.URL, gitClient, c.gitBranch)
	if err != nil {
		return u, err
	}

	repoDir := filepath.Join(ghPagesDir, outputPath)
	err = os.MkdirAll(repoDir, 0755)
	if err != nil {
		return u, err
	}

	toFile := filepath.Join(repoDir, outputPath)
	toDir, _ := filepath.Split(toFile)
	err = os.MkdirAll(toDir, util.DefaultWritePermissions)
	if err != nil {
		return u, errors.Wrapf(err, "failed to create directory file %s", toDir)
	}
	err = ioutil.WriteFile(toFile, data, util.DefaultWritePermissions)
	if err != nil {
		return u, errors.Wrapf(err, "failed to write file %s", toFile)
	}

	u = c.generateURL(storageOrg, storageRepoName, outputPath)

	err = gitClient.Add(ghPagesDir, repoDir)
	if err != nil {
		return u, err
	}
	changes, err := gitClient.HasChanges(ghPagesDir)
	if err != nil {
		return u, err
	}
	if !changes {
		return u, nil
	}
	err = gitClient.CommitDir(ghPagesDir, fmt.Sprintf("Publishing files for path %s", outputPath))
	if err != nil {
		return u, err
	}
	err = gitClient.Push(ghPagesDir)
	return u, err
}

func (c *GitCollector) generateURL(storageOrg string, storageRepoName string, rPath string) string {
	// TODO only supporting github for now!!!
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", storageOrg, storageRepoName, c.gitBranch, rPath)
	log.Infof("Publishing %s\n", util.ColorInfo(url))
	return url
}

// cloneGitHubPagesBranchToTempDir clones the github pages branch to a temp dir
func cloneGitHubPagesBranchToTempDir(sourceURL string, gitClient gits.Gitter, branchName string) (string, error) {
	// First clone the git repo
	ghPagesDir, err := ioutil.TempDir("", "jenkins-x-collect")
	if err != nil {
		return ghPagesDir, err
	}

	err = gitClient.ShallowCloneBranch(sourceURL, branchName, ghPagesDir)
	if err != nil {
		log.Infof("error doing shallow clone of branch %s: %v", branchName, err)
		// swallow the error
		log.Infof("No existing %s branch so creating it\n", branchName)
		// branch doesn't exist, so we create it following the process on https://help.github.com/articles/creating-project-pages-using-the-command-line/
		err = gitClient.Clone(sourceURL, ghPagesDir)
		if err != nil {
			return ghPagesDir, err
		}
		err = gitClient.CheckoutOrphan(ghPagesDir, branchName)
		if err != nil {
			return ghPagesDir, err
		}
		err = gitClient.RemoveForce(ghPagesDir, ".")
		if err != nil {
			return ghPagesDir, err
		}
		err = os.Remove(filepath.Join(ghPagesDir, ".gitignore"))
		if err != nil {
			// Swallow the error, doesn't matter
		}
	}
	return ghPagesDir, nil
}
