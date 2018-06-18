package gits

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"

	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

const (
	replaceInvalidBranchChars = '_'
)

// FindGitConfigDir tries to find the `.git` directory either in the current directory or in parent directories
func FindGitConfigDir(dir string) (string, string, error) {
	d := dir
	var err error
	if d == "" {
		d, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}
	for {
		gitDir := filepath.Join(d, ".git/config")
		f, err := util.FileExists(gitDir)
		if err != nil {
			return "", "", err
		}
		if f {
			return d, gitDir, nil
		}
		dirPath := strings.TrimSuffix(d, "/")
		if dirPath == "" {
			return "", "", nil
		}
		p, _ := filepath.Split(dirPath)
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
		return fmt.Errorf("failed to invoke git clone to %s due to %s", directory, err)
	}
	return nil
}

// GitCloneOrPull will clone the given git URL or pull if it alreasy exists
func GitCloneOrPull(url string, directory string) error {
	empty, err := util.IsEmpty(directory)
	if err != nil {
		return err
	}

	if !empty {
		e := exec.Command("git", "pull")
		e.Dir = directory
		e.Stdout = os.Stdout
		e.Stderr = os.Stderr
		err = e.Run()
		if err != nil {
			return fmt.Errorf("failed to git pull in %s due to %s", directory, err)
		}
		return nil
	}
	e := exec.Command("git", "clone", url, directory)
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err = e.Run()
	if err != nil {
		return fmt.Errorf("failed to git clone to %s due to %s", directory, err)
	}
	return nil
}

// CheckoutRemoteBranch checks out the given remote tracking branch
func CheckoutRemoteBranch(dir string, branch string) error {
	remoteBranch := "origin/" + branch
	remoteBranches, err := GitGetRemoteBranches(dir)
	if err != nil {
		return err
	}
	if util.StringArrayIndex(remoteBranches, remoteBranch) < 0 {
		return util.RunCommand(dir, "git", "checkout", "-t", remoteBranch)
	}
	cur, err := GitGetBranch(dir)
	if err != nil {
		return err
	}
	if cur == branch {
		return nil
	}
	return GitCheckout(dir, branch)
}

// GitGetRemoteBranches returns the remote branches
func GitGetRemoteBranches(dir string) ([]string, error) {
	answer := []string{}
	text, err := util.GetCommandOutput(dir, "git", "branch", "-r")
	if err != nil {
		return answer, err
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		columns := strings.Split(line, " ")
		for _, col := range columns {
			if col != "" {
				answer = append(answer, col)
				break
			}
		}
	}
	return answer, nil
}

// GitCheckout checks out the given branch
func GitCheckout(dir string, branch string) error {
	return util.RunCommand(dir, "git", "checkout", branch)
}

func GitInit(dir string) error {
	e := exec.Command("git", "init")
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("failed to invoke git init in %s due to %s", dir, err)
	}
	return nil
}

func GitRemove(dir, fileName string) error {
	e := exec.Command("git", "rm", "-r", fileName)
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("failed to invoke git rm in %s due to %s", dir, err)
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
		return fmt.Errorf("failed to invoke git status in %s due to %s", dir, err)
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
		return fmt.Errorf("failed to invoke git push in %s due to %s", dir, err)
	}
	return nil
}

func GitForcePushBranch(dir string, localBranch string, remoteBranch string) error {
	e := exec.Command("git", "push", "-f", "origin", localBranch+":"+remoteBranch)
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("failed to invoke git push in %s from local branch %s to remote branch %s due to %s", dir, localBranch, remoteBranch, err)
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
		return fmt.Errorf("failed to run git add in %s due to %s", dir, err)
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
	return GitCommitDir(dir, message)
}

func GitCommitDir(dir string, message string) error {
	e := exec.Command("git", "commit", "-m", message)
	e.Dir = dir
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		return fmt.Errorf("failed to run git commit in %s due to %s", dir, err)
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
		return fmt.Errorf("failed to invoke git %s in %s due to %s", strings.Join(args, " "), dir, err)
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
	repo, err := GetGitInfo(dir)
	if err != nil {
		return "", err
	}
	return repo.HostURL(), err
}

func GetGitInfo(dir string) (*GitRepositoryInfo, error) {
	e := exec.Command("git", "status")
	if dir != "" {
		e.Dir = dir
	}
	data, err := e.CombinedOutput()
	if err != nil && strings.Contains(string(data), "Not a git repository") {
		return nil, fmt.Errorf("you are not in a Git repository - promotion command should be executed from an application directory")
	}

	e = exec.Command("git", "config", "--get", "remote.origin.url")
	if dir != "" {
		e.Dir = dir
	}
	data, err = e.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run git commit in %s due to %s", dir, err)
	}

	rUrl := string(data)
	rUrl = strings.TrimSpace(rUrl)

	repo, err := ParseGitURL(rUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse git URL %s due to %s", rUrl, err)
	}
	return repo, err
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

func GetAuthorEmailForCommit(dir string, sha string) (string, error) {
	e := exec.Command("git", "show", "-s", "--format=%aE", sha)
	e.Dir = dir
	out, err := e.Output()
	if err != nil {
		return "", fmt.Errorf("failed to invoke git %s in %s due to %s", "show "+sha, dir, err)
	}

	return strings.TrimSpace(string(out)), nil
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

func parseGitConfig(gitConf string) (*gitcfg.Config, error) {
	if gitConf == "" {
		return nil, fmt.Errorf("no GitConfDir defined")
	}
	cfg := gitcfg.NewConfig()
	data, err := ioutil.ReadFile(gitConf)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s due to %s", gitConf, err)
	}

	err = cfg.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s due to %s", gitConf, err)
	}
	return cfg, nil
}

func DiscoverRemoteGitURL(gitConf string) (string, error) {
	cfg, err := parseGitConfig(gitConf)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return "", nil
	}
	rUrl := GetRemoteUrl(cfg, "origin")
	if rUrl == "" {
		rUrl = GetRemoteUrl(cfg, "upstream")
	}
	return rUrl, nil
}

func DiscoverUpstreamGitURL(gitConf string) (string, error) {
	cfg, err := parseGitConfig(gitConf)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return "", nil
	}
	rUrl := GetRemoteUrl(cfg, "upstream")
	if rUrl == "" {
		rUrl = GetRemoteUrl(cfg, "origin")
	}
	return rUrl, nil
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

func GetPreviousGitTagSHA(dir string) (string, error) {
	// when in a release branch we need to skip 2 rather that 1 to find the revision of the previous tag
	// no idea why! :)
	return util.GetCommandOutput(dir, "git", "rev-list", "--tags", "--skip=2", "--max-count=1")
}

// GetRevisionBeforeDate returns the revision before the given date
func GetRevisionBeforeDate(dir string, t time.Time) (string, error) {
	dateText := FormatDate(t)
	return GetRevisionBeforeDateText(dir, dateText)
}

// GetRevisionBeforeDateText returns the revision before the given date in format "MonthName dayNumber year"
func GetRevisionBeforeDateText(dir string, dateText string) (string, error) {
	branch, err := GitGetBranch(dir)
	if err != nil {
		return "", err
	}
	return util.GetCommandOutput(dir, "git", "rev-list", "-1", "--before=\""+dateText+"\"", "--max-count=1", branch)
}

func FormatDate(t time.Time) string {
	return fmt.Sprintf("%s %d %d", t.Month().String(), t.Day(), t.Year())
}

func ParseDate(dateText string) (time.Time, error) {
	return time.Parse(DateFormat, dateText)
}

func GetCurrentGitTagSHA(dir string) (string, error) {
	return util.GetCommandOutput(dir, "git", "rev-list", "--tags", "--max-count=1")
}

func PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer) {
	tokenUrl := ProviderAccessTokenURL(server.Kind, server.URL, username)

	fmt.Fprintf(o, "To be able to create a repository on %s we need an API Token\n", server.Label())
	fmt.Fprintf(o, "Please click this URL %s\n\n", util.ColorInfo(tokenUrl))
	fmt.Fprint(o, "Then COPY the token and enter in into the form below:\n\n")
}

func GitIsFork(gitProvider GitProvider, gitInfo *GitRepositoryInfo, dir string) (bool, error) {
	// lets ignore errors as that just means there's no config
	originUrl, _ := util.GetCommandOutput(dir, "git", "config", "--get", "remote.origin.url")
	upstreamUrl, _ := util.GetCommandOutput(dir, "git", "config", "--get", "remote.upstream.url")

	if originUrl != upstreamUrl && originUrl != "" && upstreamUrl != "" {
		return true, nil
	}

	repo, err := gitProvider.GetRepository(gitInfo.Organisation, gitInfo.Name)
	if err != nil {
		return false, err
	}
	return repo.Fork, nil
}

// ToGitLabels converts the list of label names into an array of GitLabels
func ToGitLabels(names []string) []GitLabel {
	answer := []GitLabel{}
	for _, n := range names {
		answer = append(answer, GitLabel{Name: n})
	}
	return answer
}

// GetHost returns the Git Provider hostname, e.g github.com
func GetHost(gitProvider GitProvider) (string, error) {
	if gitProvider == nil {
		return "", fmt.Errorf("no git provider")
	}

	if gitProvider.ServerURL() == "" {
		return "", fmt.Errorf("no git provider server URL found")
	}
	url, err := url.Parse(gitProvider.ServerURL())
	if err != nil {
		return "", fmt.Errorf("error parsing ")
	}
	return url.Host, nil
}
