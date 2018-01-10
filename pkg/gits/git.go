package gits

import (
	"fmt"
	"os/exec"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/util"
	"strings"
	"github.com/jenkins-x/jx/pkg/auth"
	"net/url"
)

// FindGitConfigDir tries to find the `.git` directory either in the current directory or in parent directories
func FindGitConfigDir(dir string) (string, string, error) {
	d := dir
	for {
		gitDir := filepath.Join(d, ".git/config")
		f, err := util.FileExists(gitDir)
		if err != nil {
			return "", "", err
		}
		if f {
			return d, gitDir, nil
		}
		p, _ := filepath.Split(d)
		if d == "/" || p == d {
			return "", "", nil
		}
		d = p
	}

}

// GitClone clones the given git URL into the given directory
func GitClone(url string, directory string) error {
	/*
	return git.PlainClone(directory, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	*/
	e := exec.Command("git", "clone", url, directory)
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("Failed to invoke git init in %s due to %s", directory, err)
	}
	return nil
}

func GitInit(dir string) error {
	e := exec.Command("git", "init")
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("Failed to invoke git init in %s due to %s", dir, err)
	}
	return nil
}

func GitStatus(dir string) error {
	e := exec.Command("git", "status")
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("Failed to invoke git status in %s due to %s", dir, err)
	}
	return nil
}

func GitPush(dir string) error {
	e := exec.Command("git", "push")
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("Failed to invoke git push in %s due to %s", dir, err)
	}
	return nil
}

func GitAdd(dir string, args ...string) error {
	a := append([]string{"add"}, args...)
	e := exec.Command("git", a...)
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("Failed to run git add in %s due to %s", dir, err)
	}
	return nil
}

func HasChanges(dir string) (bool, error) {
	e := exec.Command("git", "status", "-s")
	e.Dir = dir
	data, err := e.Output()
	if err != nil {
	  return false, err
	}
	text := string(data)
	text = strings.TrimSpace(text)
	return len(text) > 0, nil
}


func GitCommitIfChanges(dir string, message string) error {
	changed, err := HasChanges(dir)
	if err != nil {
	  return err
	}
	if !changed {
		return nil
	}
	return GitCommit(dir, message)
}


func GitCommit(dir string, message string) error {
	e := exec.Command("git", "commit", "-m", message)
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("Failed to run git commit in %s due to %s", dir, err)
	}
	return nil
}

func GitRemoteAddOrigin(dir string, url string) error {

	e := exec.Command("git", "remote", "add", "origin", url)
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("Failed to invoke git status in %s due to %s", dir, err)
	}
	return nil
}


func GitCmd(dir string, args ...string) error {
	e := exec.Command("git", args...)
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("Failed to invoke git %s in %s due to %s", strings.Join(args, " "), dir, err)
	}
	return nil
}


// GitCreatePushURL creates the git repository URL with the username and password encoded for HTTPS based URLs
func GitCreatePushURL(cloneURL string, userAuth *auth.UserAuth) (string, error) {
	u, err := url.Parse(cloneURL)
	if err != nil {
		// already a git/ssh url?
		return cloneURL, nil
	}
	if userAuth.Username != "" || userAuth.ApiToken != ""{
		u.User = url.UserPassword(userAuth.Username, userAuth.ApiToken)
		return u.String(), nil
	}
	return cloneURL, nil
}


