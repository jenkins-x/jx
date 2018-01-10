package gits

import (
	"fmt"
	"os/exec"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx/pkg/util"
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
