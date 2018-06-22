package gits

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
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
	return GitCmd(directory, "clone", url)
}

// GitCloneOrPull will clone the given git URL or pull if it alreasy exists
func GitCloneOrPull(url string, directory string) error {
	empty, err := util.IsEmpty(directory)
	if err != nil {
		return err
	}

	if !empty {
		return GitCmd(directory, "pull")
	}
	return GitCmd("clone", url)
}

// CheckoutRemoteBranch checks out the given remote tracking branch
func CheckoutRemoteBranch(dir string, branch string) error {
	remoteBranch := "origin/" + branch
	remoteBranches, err := GitGetRemoteBranches(dir)
	if err != nil {
		return err
	}
	if util.StringArrayIndex(remoteBranches, remoteBranch) < 0 {
		return GitCmd(dir, "checkout", "-t", remoteBranch)
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
	text, err := GitCmdWithOutput(dir, "branch", "-r")
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
	return GitCmd(dir, "checkout", branch)
}

func GitInit(dir string) error {
	return GitCmd(dir, "init")
}

func GitRemove(dir, fileName string) error {
	return GitCmd(dir, "rm", "-r", fileName)
}

func GitStatus(dir string) error {
	return GitCmd(dir, "status")
}

func GitGetBranch(dir string) (string, error) {
	return GitCmdWithOutput(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

func GitPush(dir string) error {
	return GitCmd(dir, "push", "origin", "HEAD")
}

func GitForcePushBranch(dir string, localBranch string, remoteBranch string) error {
	return GitCmd(dir, "push", "-f", "origin", localBranch+":"+remoteBranch)
}

func GitAdd(dir string, args ...string) error {
	add := append([]string{"add"}, args...)
	return GitCmd(dir, add...)
}

func HasChanges(dir string) (bool, error) {
	text, err := GitCmdWithOutput(dir, "status", "-s")
	if err != nil {
		return false, err
	}
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
	return GitCmd(dir, "commit", "-m", message)
}

func GitCmd(dir string, args ...string) error {
	return util.RunCommand(dir, "git", args...)
}

func GitCmdWithOutput(dir string, args ...string) (string, error) {
	return util.GetCommandOutput(dir, "git", args...)
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
	text, err := GitCmdWithOutput(dir, "status")
	if err != nil && strings.Contains(text, "Not a git repository") {
		return nil, fmt.Errorf("you are not in a Git repository - promotion command should be executed from an application directory")
	}

	text, err = GitCmdWithOutput(dir, "config", "--get", "remote.origin.url")
	rUrl := strings.TrimSpace(text)

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
	text, err := GitCmdWithOutput(dir, "show", "-s", "--format=%aE", sha)
	if err != nil {
		return "", fmt.Errorf("failed to invoke git %s in %s due to %s", "show "+sha, dir, err)
	}

	return strings.TrimSpace(text), nil
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
	text, err := GitCmdWithOutput(dir, "branch", "-a")
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
	return GitCmdWithOutput(dir, "rev-list", "--tags", "--skip=2", "--max-count=1")
}

// GetRevisionBeforeDate returns the revision before the given date
func GetRevisionBeforeDate(dir string, t time.Time) (string, error) {
	dateText := util.FormatDate(t)
	return GetRevisionBeforeDateText(dir, dateText)
}

// GetRevisionBeforeDateText returns the revision before the given date in format "MonthName dayNumber year"
func GetRevisionBeforeDateText(dir string, dateText string) (string, error) {
	branch, err := GitGetBranch(dir)
	if err != nil {
		return "", err
	}
	return GitCmdWithOutput(dir, "rev-list", "-1", "--before=\""+dateText+"\"", "--max-count=1", branch)
}

func GetCurrentGitTagSHA(dir string) (string, error) {
	return GitCmdWithOutput(dir, "rev-list", "--tags", "--max-count=1")
}

func GitFetchTags(dir string) error {
	return GitCmd("", "fetch", "--tags", "-v")
}

func GitTags(dir string) ([]string, error) {
	tags := []string{}
	text, err := GitCmdWithOutput(dir, "tag")
	if err != nil {
		return tags, err
	}
	text = strings.TrimSuffix(text, "\n")
	return strings.Split(text, "\n"), nil
}

func PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer) {
	tokenUrl := ProviderAccessTokenURL(server.Kind, server.URL, username)

	fmt.Fprintf(o, "To be able to create a repository on %s we need an API Token\n", server.Label())
	fmt.Fprintf(o, "Please click this URL %s\n\n", util.ColorInfo(tokenUrl))
	fmt.Fprint(o, "Then COPY the token and enter in into the form below:\n\n")
}

func GitIsFork(gitProvider GitProvider, gitInfo *GitRepositoryInfo, dir string) (bool, error) {
	// lets ignore errors as that just means there's no config
	originUrl, _ := GitCmdWithOutput(dir, "config", "--get", "remote.origin.url")
	upstreamUrl, _ := GitCmdWithOutput(dir, "config", "--get", "remote.upstream.url")

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

func GitVersion() (string, error) {
	return GitCmdWithOutput("", "version")
}

func GitUsername(dir string) (string, error) {
	return GitCmdWithOutput("", "config", "--global", "--get", "user.name")
}

func GitSetUsername(dir string, username string) error {
	return GitCmd(dir, "config", "--global", "--add", "user.name", username)
}
func GitEmail(dir string) (string, error) {
	return GitCmdWithOutput(dir, "config", "--global", "--get", "user.email")
}
func GitSetEmail(dir string, email string) error {
	return GitCmd(dir, "config", "--global", "--add", "user.email", email)
}
