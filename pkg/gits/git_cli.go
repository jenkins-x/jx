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

type GitCLI struct{}

func NewGitCLI() *GitCLI {
	return &GitCLI{}
}

// FindGitConfigDir tries to find the `.git` directory either in the current directory or in parent directories
func (g *GitCLI) FindGitConfigDir(dir string) (string, string, error) {
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

// Clone clones the given git URL into the given directory
func (g *GitCLI) Clone(url string, directory string) error {
	/*
		return git.PlainClone(directory, false, &git.CloneOptions{
			URL:               url,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		})
	*/
	return g.gitCmd(directory, "clone", url)
}

func (g *GitCLI) Pull(dir string) error {
	return g.gitCmd(dir, "pull")
}

// CloneOrPull will clone the given git URL or pull if it alreasy exists
func (g *GitCLI) CloneOrPull(url string, directory string) error {
	empty, err := util.IsEmpty(directory)
	if err != nil {
		return err
	}

	if !empty {
		return g.Pull(directory)
	}
	return g.gitCmd("clone", url)
}

func (g *GitCLI) PullUpstream(dir string) error {
	return g.gitCmd(dir, "pull", "-r", "upstream", "master")
}

func (g *GitCLI) AddRemote(dir string, url string, remote string) error {
	return g.gitCmd(dir, "remote", "add", remote, url)
}

func (g *GitCLI) UpdateRemote(dir, url string) error {
	return g.gitCmd(dir, "remote", "set-url", "origin", url)
}

func (g *GitCLI) Stash(dir string) error {
	return g.gitCmd(dir, "stash")
}

// CheckoutRemoteBranch checks out the given remote tracking branch
func (g *GitCLI) CheckoutRemoteBranch(dir string, branch string) error {
	remoteBranch := "origin/" + branch
	remoteBranches, err := g.RemoteBranches(dir)
	if err != nil {
		return err
	}
	if util.StringArrayIndex(remoteBranches, remoteBranch) < 0 {
		return g.gitCmd(dir, "checkout", "-t", remoteBranch)
	}
	cur, err := g.Branch(dir)
	if err != nil {
		return err
	}
	if cur == branch {
		return nil
	}
	return g.Checkout(dir, branch)
}

// RemoteBranches returns the remote branches
func (g *GitCLI) RemoteBranches(dir string) ([]string, error) {
	answer := []string{}
	text, err := g.gitCmdWithOutput(dir, "branch", "-r")
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

// Checkout checks out the given branch
func (g *GitCLI) Checkout(dir string, branch string) error {
	return g.gitCmd(dir, "checkout", branch)
}

func (g *GitCLI) Init(dir string) error {
	return g.gitCmd(dir, "init")
}

func (g *GitCLI) Remove(dir, fileName string) error {
	return g.gitCmd(dir, "rm", "-r", fileName)
}

func (g *GitCLI) Status(dir string) error {
	return g.gitCmd(dir, "status")
}

func (g *GitCLI) Branch(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

func (g *GitCLI) Push(dir string) error {
	return g.gitCmd(dir, "push", "origin", "HEAD")
}

func (g *GitCLI) ForcePushBranch(dir string, localBranch string, remoteBranch string) error {
	return g.gitCmd(dir, "push", "-f", "origin", localBranch+":"+remoteBranch)
}

func (g *GitCLI) PushMaster(dir string) error {
	return g.gitCmd(dir, "push", "-u", "origin", "master")
}

func (g *GitCLI) PushTag(dir string, tag string) error {
	return g.gitCmd(dir, "push", "origin", tag)
}

func (g *GitCLI) Add(dir string, args ...string) error {
	add := append([]string{"add"}, args...)
	return g.gitCmd(dir, add...)
}

func (g *GitCLI) HasChanges(dir string) (bool, error) {
	text, err := g.gitCmdWithOutput(dir, "status", "-s")
	if err != nil {
		return false, err
	}
	text = strings.TrimSpace(text)
	return len(text) > 0, nil
}

func (g *GitCLI) CommitIfChanges(dir string, message string) error {
	changed, err := g.HasChanges(dir)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return g.CommitDir(dir, message)
}

func (g *GitCLI) CommitDir(dir string, message string) error {
	return g.gitCmd(dir, "commit", "-m", message)
}

func (g *GitCLI) AddCommmit(dir string, msg string) error {
	return g.gitCmd("", "commit", "-a", "-m", msg, "--allow-empty")
}

func (g *GitCLI) gitCmd(dir string, args ...string) error {
	return util.RunCommand(dir, "git", args...)
}

func (g *GitCLI) gitCmdWithOutput(dir string, args ...string) (string, error) {
	return util.GetCommandOutput(dir, "git", args...)
}

// CreatePushURL creates the git repository URL with the username and password encoded for HTTPS based URLs
func (g *GitCLI) CreatePushURL(cloneURL string, userAuth *auth.UserAuth) (string, error) {
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

func (g *GitCLI) RepoName(org, repoName string) string {
	if org != "" {
		return org + "/" + repoName
	}
	return repoName
}

func (g *GitCLI) Server(dir string) (string, error) {
	repo, err := g.Info(dir)
	if err != nil {
		return "", err
	}
	return repo.HostURL(), err
}

func (g *GitCLI) Info(dir string) (*GitRepositoryInfo, error) {
	text, err := g.gitCmdWithOutput(dir, "status")
	if err != nil && strings.Contains(text, "Not a git repository") {
		return nil, fmt.Errorf("you are not in a Git repository - promotion command should be executed from an application directory")
	}

	text, err = g.gitCmdWithOutput(dir, "config", "--get", "remote.origin.url")
	rUrl := strings.TrimSpace(text)

	repo, err := ParseGitURL(rUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse git URL %s due to %s", rUrl, err)
	}
	return repo, err
}

// ConvertToValidBranchName converts the given branch name into a valid git branch string
// replacing any dodgy characters
func (g *GitCLI) ConvertToValidBranchName(name string) string {
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

func (g *GitCLI) GetAuthorEmailForCommit(dir string, sha string) (string, error) {
	text, err := g.gitCmdWithOutput(dir, "show", "-s", "--format=%aE", sha)
	if err != nil {
		return "", fmt.Errorf("failed to invoke git %s in %s due to %s", "show "+sha, dir, err)
	}

	return strings.TrimSpace(text), nil
}

func (g *GitCLI) SetRemoteURL(dir string, name string, gitURL string) error {
	err := g.gitCmd(dir, "remote", "add", name, gitURL)
	if err != nil {
		err = g.gitCmd(dir, "remote", "set-url", name, gitURL)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *GitCLI) parseGitConfig(gitConf string) (*gitcfg.Config, error) {
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

func (g *GitCLI) DiscoverRemoteGitURL(gitConf string) (string, error) {
	cfg, err := g.parseGitConfig(gitConf)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return "", nil
	}
	rUrl := g.GetRemoteUrl(cfg, "origin")
	if rUrl == "" {
		rUrl = g.GetRemoteUrl(cfg, "upstream")
	}
	return rUrl, nil
}

func (g *GitCLI) DiscoverUpstreamGitURL(gitConf string) (string, error) {
	cfg, err := g.parseGitConfig(gitConf)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return "", nil
	}
	rUrl := g.GetRemoteUrl(cfg, "upstream")
	if rUrl == "" {
		rUrl = g.GetRemoteUrl(cfg, "origin")
	}
	return rUrl, nil
}

func (g *GitCLI) firstRemoteUrl(remote *gitcfg.RemoteConfig) string {
	if remote != nil {
		urls := remote.URLs
		if urls != nil && len(urls) > 0 {
			return urls[0]
		}
	}
	return ""
}

func (g *GitCLI) GetRemoteUrl(config *gitcfg.Config, name string) string {
	if config.Remotes != nil {
		return g.firstRemoteUrl(config.Remotes[name])
	}
	return ""
}

func (g *GitCLI) RemoteBranchNames(dir string, prefix string) ([]string, error) {
	answer := []string{}
	text, err := g.gitCmdWithOutput(dir, "branch", "-a")
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

func (g *GitCLI) GetPreviousGitTagSHA(dir string) (string, error) {
	// when in a release branch we need to skip 2 rather that 1 to find the revision of the previous tag
	// no idea why! :)
	return g.gitCmdWithOutput(dir, "rev-list", "--tags", "--skip=2", "--max-count=1")
}

// GetRevisionBeforeDate returns the revision before the given date
func (g *GitCLI) GetRevisionBeforeDate(dir string, t time.Time) (string, error) {
	dateText := util.FormatDate(t)
	return g.GetRevisionBeforeDateText(dir, dateText)
}

// GetRevisionBeforeDateText returns the revision before the given date in format "MonthName dayNumber year"
func (g *GitCLI) GetRevisionBeforeDateText(dir string, dateText string) (string, error) {
	branch, err := g.Branch(dir)
	if err != nil {
		return "", err
	}
	return g.gitCmdWithOutput(dir, "rev-list", "-1", "--before=\""+dateText+"\"", "--max-count=1", branch)
}

func (g *GitCLI) GetCurrentGitTagSHA(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "rev-list", "--tags", "--max-count=1")
}

func (g *GitCLI) FetchTags(dir string) error {
	return g.gitCmd("", "fetch", "--tags", "-v")
}

func (g *GitCLI) Tags(dir string) ([]string, error) {
	tags := []string{}
	text, err := g.gitCmdWithOutput(dir, "tag")
	if err != nil {
		return tags, err
	}
	text = strings.TrimSuffix(text, "\n")
	return strings.Split(text, "\n"), nil
}

func (g *GitCLI) CreateTag(dir string, tag string, msg string) error {
	return g.gitCmd("", "tag", "-fa", tag, "-m", msg)
}

func (g *GitCLI) PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer) {
	tokenUrl := ProviderAccessTokenURL(server.Kind, server.URL, username)

	fmt.Fprintf(o, "To be able to create a repository on %s we need an API Token\n", server.Label())
	fmt.Fprintf(o, "Please click this URL %s\n\n", util.ColorInfo(tokenUrl))
	fmt.Fprint(o, "Then COPY the token and enter in into the form below:\n\n")
}

func (g *GitCLI) IsFork(gitProvider GitProvider, gitInfo *GitRepositoryInfo, dir string) (bool, error) {
	// lets ignore errors as that just means there's no config
	originUrl, _ := g.gitCmdWithOutput(dir, "config", "--get", "remote.origin.url")
	upstreamUrl, _ := g.gitCmdWithOutput(dir, "config", "--get", "remote.upstream.url")

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
func (g *GitCLI) ToGitLabels(names []string) []GitLabel {
	answer := []GitLabel{}
	for _, n := range names {
		answer = append(answer, GitLabel{Name: n})
	}
	return answer
}

func (g *GitCLI) Version() (string, error) {
	return g.gitCmdWithOutput("", "version")
}

func (g *GitCLI) Username(dir string) (string, error) {
	return g.gitCmdWithOutput("", "config", "--global", "--get", "user.name")
}

func (g *GitCLI) SetUsername(dir string, username string) error {
	return g.gitCmd(dir, "config", "--global", "--add", "user.name", username)
}
func (g *GitCLI) Email(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "config", "--global", "--get", "user.email")
}
func (g *GitCLI) SetEmail(dir string, email string) error {
	return g.gitCmd(dir, "config", "--global", "--add", "user.email", email)
}

func (g *GitCLI) CreateBranch(dir string, branch string) error {
	return g.gitCmd(dir, "branch", branch)

}
