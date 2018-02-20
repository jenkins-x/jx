package gits

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"

	"io"

	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

const (
	replaceInvalidBranchChars = '_'
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

func GitGetBranch(dir string) (string, error) {
	return util.GetCommandOutput(dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
}

func GitPush(dir string) error {
	e := exec.Command("git", "push", "origin", "HEAD")
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
	if userAuth.Username != "" || userAuth.ApiToken != "" {
		u.User = url.UserPassword(userAuth.Username, userAuth.ApiToken)
		return u.String(), nil
	}
	return cloneURL, nil
}

func GitRepoName(org, repoName string) string {
	if org != "" {
		return org + "/" + repoName
	}
	return repoName
}

func GetGitServer(dir string) (string, error) {
	e := exec.Command("git", "config", "--get", "remote.origin.url")
	if dir != "" {
		e.Dir = dir
	}
	data, err := e.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to run git commit in %s due to %s", dir, err)
	}

	url := string(data)
	url = strings.TrimSpace(url)

	repo, err := ParseGitURL(url)
	if err != nil {
		return "", fmt.Errorf("Failed to parse git URL %s due to %s", url, err)
	}

	return repo.Host, err
}

// ConvertToValidBranchName converts the given branch name into a valid git branch string
// replacing any dodgy characters
func ConvertToValidBranchName(name string) string {
	name = strings.TrimSuffix(name, "/")
	name = strings.TrimSuffix(name, ".lock")
	var buffer bytes.Buffer

	last := ' '
	for _, ch := range name {
		if ch <= 32 {
			ch = replaceInvalidBranchChars
		}
		switch ch {
		case '~':
			ch = replaceInvalidBranchChars
		case '^':
			ch = replaceInvalidBranchChars
		case ':':
			ch = replaceInvalidBranchChars
		case ' ':
			ch = replaceInvalidBranchChars
		case '\n':
			ch = replaceInvalidBranchChars
		case '\r':
			ch = replaceInvalidBranchChars
		case '\t':
			ch = replaceInvalidBranchChars
		}
		if ch != replaceInvalidBranchChars || last != replaceInvalidBranchChars {
			buffer.WriteString(string(ch))
		}
		last = ch
	}
	return buffer.String()
}

func SetRemoteURL(dir string, name string, gitURL string) error {
	err := GitCmd(dir, "remote", "add", name, gitURL)
	if err != nil {
		err = GitCmd(dir, "remote", "set-url", name, gitURL)
		if err != nil {
			return err
		}
	}
	return nil
}

func DiscoverRemoteGitURL(gitConf string) (string, error) {
	if gitConf == "" {
		return "", fmt.Errorf("No GitConfDir defined!")
	}
	cfg := gitcfg.NewConfig()
	data, err := ioutil.ReadFile(gitConf)
	if err != nil {
		return "", fmt.Errorf("Failed to load %s due to %s", gitConf, err)
	}

	err = cfg.Unmarshal(data)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return "", nil
	}
	url := GetRemoteUrl(cfg, "origin")
	if url == "" {
		url = GetRemoteUrl(cfg, "upstream")
	}
	return url, nil
}

func firstRemoteUrl(remote *gitcfg.RemoteConfig) string {
	if remote != nil {
		urls := remote.URLs
		if urls != nil && len(urls) > 0 {
			return urls[0]
		}
	}
	return ""
}

func GetRemoteUrl(config *gitcfg.Config, name string) string {
	if config.Remotes != nil {
		return firstRemoteUrl(config.Remotes[name])
	}
	return ""
}

func GitGetRemoteBranchNames(dir string, prefix string) ([]string, error) {
	answer := []string{}
	text, err := util.GetCommandOutput(dir, "git", "branch", "-a")
	if err != nil {
		return answer, err
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "* "))
		if prefix != "" {
			line = strings.TrimPrefix(line, prefix)
		}
		answer = append(answer, line)
	}
	return answer, nil
}

func PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, o io.Writer) {
	tokenUrl := ProviderAccessTokenURL(server.Kind, server.URL)

	fmt.Fprintf(o, "To be able to create a repository on %s we need an API Token\n", server.Label())
	fmt.Fprintf(o, "Please click this URL %s\n\n", util.ColorInfo(tokenUrl))
	fmt.Fprintf(o, "Then COPY the token and enter in into the form below:\n\n")
}
