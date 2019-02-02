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

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"

	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

const (
	replaceInvalidBranchChars = '_'
)

// GitCLI implements common git actions based on git CLI
type GitCLI struct{}

// NewGitCLI creates a new GitCLI instance
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
func (g *GitCLI) Clone(url string, dir string) error {
	return g.gitCmd(dir, "clone", url, ".")
}

// Clone clones a single branch of the given git URL into the given directory
func (g *GitCLI) ShallowCloneBranch(url string, branch string, dir string) error {
	return g.gitCmd(dir, "clone", "--depth", "1", "--single-branch", "--branch", branch, url, ".")
}

// Pull pulls the Git repository in the given directory
func (g *GitCLI) Pull(dir string) error {
	return g.gitCmd(dir, "pull")
}

// PullRemoteBranches pulls the remote Git tags from the given given directory
func (g *GitCLI) PullRemoteBranches(dir string) error {
	return g.gitCmd(dir, "pull", "--all")
}

// DeleteRemoteBranch deletes the remote branch in the given given directory
func (g *GitCLI) DeleteRemoteBranch(dir string, remoteName string, branch string) error {
	return g.gitCmd(dir, "push", remoteName, "--delete", branch)
}

// CloneOrPull clones  the given git URL or pull if it already exists
func (g *GitCLI) CloneOrPull(url string, dir string) error {
	empty, err := util.IsEmpty(dir)
	if err != nil {
		return err
	}

	if !empty {
		return g.Pull(dir)
	}
	return g.Clone(url, dir)
}

// PullUpstream pulls the remote upstream branch into master branch into the given directory
func (g *GitCLI) PullUpstream(dir string) error {
	return g.gitCmd(dir, "pull", "-r", "upstream", "master")
}

// AddRemote adds a remote repository at the given URL and with the given name
func (g *GitCLI) AddRemote(dir string, name string, url string) error {
	return g.gitCmd(dir, "remote", "add", name, url)
}

// UpdateRemote updates the URL of the remote repository
func (g *GitCLI) UpdateRemote(dir, url string) error {
	return g.gitCmd(dir, "remote", "set-url", "origin", url)
}

// Stash stashes the current changes from the given directory
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

// Checkout checks out the given branch
func (g *GitCLI) CheckoutOrphan(dir string, branch string) error {
	return g.gitCmd(dir, "checkout", "--orphan", branch)
}

// Init inits a git repository into the given directory
func (g *GitCLI) Init(dir string) error {
	return g.gitCmd(dir, "init")
}

// Remove removes the given file from a Git repository located at the given directory
func (g *GitCLI) Remove(dir, fileName string) error {
	return g.gitCmd(dir, "rm", "-r", fileName)
}

// Remove force removes the given file from a git repository located at the given directory
func (g *GitCLI) RemoveForce(dir, fileName string) error {
	return g.gitCmd(dir, "rm", "-rf", fileName)
}

// Clean force cleans a git repository located at a given directory
func (g *GitCLI) CleanForce(dir, fileName string) error {
	return g.gitCmd(dir, "clean", "-fd", fileName)
}

// Status returns the status of the git repository at the given directory
func (g *GitCLI) Status(dir string) error {
	return g.gitCmd(dir, "status")
}

// Branch returns the current branch of the repository located at the given directory
func (g *GitCLI) Branch(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

// WriteOperation performs a generic write operation, with nicer error handling
func (g *GitCLI) WriteOperation(dir string, args ...string) error {
	return errors.Wrap(g.gitCmd(dir, args...),
		"Have you set up a git credential helper? See https://help.github.com/articles/caching-your-github-password-in-git/\n")
}

// Push pushes the changes from the repository at the given directory
func (g *GitCLI) Push(dir string) error {
	return g.WriteOperation(dir, "push", "origin", "HEAD")
}

// ForcePushBranch does a force push of the local branch into the remote branch of the repository at the given directory
func (g *GitCLI) ForcePushBranch(dir string, localBranch string, remoteBranch string) error {
	return g.WriteOperation(dir, "push", "-f", "origin", localBranch+":"+remoteBranch)
}

// PushMaster pushes the master branch into the origin
func (g *GitCLI) PushMaster(dir string) error {
	return g.WriteOperation(dir, "push", "-u", "origin", "master")
}

// Pushtag pushes the given tag into the origin
func (g *GitCLI) PushTag(dir string, tag string) error {
	return g.WriteOperation(dir, "push", "origin", tag)
}

// Add does a git add for all the given arguments
func (g *GitCLI) Add(dir string, args ...string) error {
	add := append([]string{"add"}, args...)
	return g.gitCmd(dir, add...)
}

// HasChanges indicates if there are any changes in the repository from the given directory
func (g *GitCLI) HasChanges(dir string) (bool, error) {
	text, err := g.gitCmdWithOutput(dir, "status", "-s")
	if err != nil {
		return false, err
	}
	text = strings.TrimSpace(text)
	return len(text) > 0, nil
}

// CommiIfChanges does a commit if there are any changes in the repository at the given directory
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

// CommitDir commits all changes from the given directory
func (g *GitCLI) CommitDir(dir string, message string) error {
	return g.gitCmd(dir, "commit", "-m", message)
}

// AddCommit perform an add and commit of the changes from the repository at the given directory with the given messages
func (g *GitCLI) AddCommit(dir string, msg string) error {
	return g.gitCmd(dir, "commit", "-a", "-m", msg, "--allow-empty")
}

func (g *GitCLI) gitCmd(dir string, args ...string) error {
	cmd := util.Command{
		Dir:  dir,
		Name: "git",
		Args: args,
	}
	_, err := cmd.RunWithoutRetry()
	return err
}

func (g *GitCLI) gitCmdWithOutput(dir string, args ...string) (string, error) {
	cmd := util.Command{
		Dir:  dir,
		Name: "git",
		Args: args,
	}
	return cmd.RunWithoutRetry()
}

// CreatePushURL creates the Git repository URL with the username and password encoded for HTTPS based URLs
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

// RepoName formats the repository names based on the organization
func (g *GitCLI) RepoName(org, repoName string) string {
	if org != "" {
		return org + "/" + repoName
	}
	return repoName
}

// Server returns the Git server of the repository at the given directory
func (g *GitCLI) Server(dir string) (string, error) {
	repo, err := g.Info(dir)
	if err != nil {
		return "", err
	}
	return repo.HostURL(), err
}

// Info returns the git info of the repository at the given directory
func (g *GitCLI) Info(dir string) (*GitRepository, error) {
	text, err := g.gitCmdWithOutput(dir, "status")
	var rUrl string
	if err != nil && strings.Contains(text, "Not a git repository") {
		rUrl = os.Getenv("SOURCE_URL")
		if rUrl == "" {
			// Relevant in a Jenkins pipeline triggered by a PR
			rUrl = os.Getenv("CHANGE_URL")
			if rUrl == "" {
				return nil, fmt.Errorf("you are not in a Git repository - promotion command should be executed from an application directory")
			}
		}
	} else {
		text, err = g.gitCmdWithOutput(dir, "config", "--get", "remote.origin.url")
		rUrl = strings.TrimSpace(text)
	}

	repo, err := ParseGitURL(rUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Git URL %s due to %s", rUrl, err)
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

func (g *GitCLI) FetchBranch(dir string, repo string, refspec string) error {
	return g.gitCmd(dir, "fetch", repo, refspec)
}

// GetAuthorEmailForCommit returns the author email from commit message with the given SHA
func (g *GitCLI) GetAuthorEmailForCommit(dir string, sha string) (string, error) {
	text, err := g.gitCmdWithOutput(dir, "show", "-s", "--format=%aE", sha)
	if err != nil {
		return "", fmt.Errorf("failed to invoke git %s in %s due to %s", "show "+sha, dir, err)
	}

	return strings.TrimSpace(text), nil
}

// SetRemoteURL sets the remote URL of the remote with the given name
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

// DiscoverRemoteGitURL discovers the remote git URL from the given git configuration
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

// DiscoverUpstreamGitURL discovers the upstream git URL from the given git configuration
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

// GetRemoteUrl returns the remote URL from the given git config
func (g *GitCLI) GetRemoteUrl(config *gitcfg.Config, name string) string {
	if config.Remotes != nil {
		return g.firstRemoteUrl(config.Remotes[name])
	}
	return ""
}

// RemoteBranches returns all remote branches with the given prefix
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

// GetPreviousGitTagSHA returns the previous git tag from the repository at the given directory
func (g *GitCLI) GetPreviousGitTagSHA(dir string) (string, error) {
	latestTag, err := g.gitCmdWithOutput(dir, "describe", "--abbrev=0", "--tags", "--always")
	if err != nil {
		return "", fmt.Errorf("failed to find latest tag for project in %s : %s", dir, err)
	}

	previousTag, err := g.gitCmdWithOutput(dir, "describe", "--abbrev=0", "--tags", "--always", latestTag+"^")
	if err != nil {
		return "", fmt.Errorf("failed to find previous tag for project in %s : %s", dir, err)
	}
	return previousTag, err
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

// GetCurrentGitTagSHA return the SHA of the current git tag from the repository at the given directory
func (g *GitCLI) GetCurrentGitTagSHA(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "rev-list", "--tags", "--max-count=1")
}

// GetLatestCommitMessage returns the latest git commit message
func (g *GitCLI) GetLatestCommitMessage(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "log", "-1", "--pretty=%B")
}

// FetchTags fetches all the tags
func (g *GitCLI) FetchTags(dir string) error {
	return g.gitCmd("", "fetch", "--tags", "-v")
}

// Tags returns all tags from the repository at the given directory
func (g *GitCLI) Tags(dir string) ([]string, error) {
	tags := []string{}
	text, err := g.gitCmdWithOutput(dir, "tag")
	if err != nil {
		return tags, err
	}
	text = strings.TrimSuffix(text, "\n")
	return strings.Split(text, "\n"), nil
}

// CreateTag creates a tag with the given name and message in the repository at the given directory
func (g *GitCLI) CreateTag(dir string, tag string, msg string) error {
	return g.gitCmd("", "tag", "-fa", tag, "-m", msg)
}

// PrintCreateRepositoryGenerateAccessToken prints the access token URL of a Git repository
func (g *GitCLI) PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer) {
	tokenUrl := ProviderAccessTokenURL(server.Kind, server.URL, username)

	fmt.Fprintf(o, "To be able to create a repository on %s we need an API Token\n", server.Label())
	fmt.Fprintf(o, "Please click this URL %s\n\n", util.ColorInfo(tokenUrl))
	fmt.Fprint(o, "Then COPY the token and enter in into the form below:\n\n")
}

// IsFork indicates if the repository at the given directory is a fork
func (g *GitCLI) IsFork(dir string) (bool, error) {
	// lets ignore errors as that just means there's no config
	originUrl, _ := g.gitCmdWithOutput(dir, "config", "--get", "remote.origin.url")
	upstreamUrl, _ := g.gitCmdWithOutput(dir, "config", "--get", "remote.upstream.url")

	if originUrl != upstreamUrl && originUrl != "" && upstreamUrl != "" {
		return true, nil
	}
	return false, fmt.Errorf("could not confirm the repo is a fork")
}

// Version returns the git version
func (g *GitCLI) Version() (string, error) {
	return g.gitCmdWithOutput("", "version")
}

// Username return the username from the git configuration
func (g *GitCLI) Username(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "config", "--global", "--get", "user.name")
}

// SetUsername sets the username in the git configuration
func (g *GitCLI) SetUsername(dir string, username string) error {
	// Will return status 1 silently if the user is not set.
	_, err := g.gitCmdWithOutput(dir, "config", "--global", "--get", "user.name")
	if err != nil {
		return g.gitCmd(dir, "config", "--global", "--add", "user.name", username)
	}
	return nil
}

// Email returns the email from the git configuration
func (g *GitCLI) Email(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "config", "--global", "--get", "user.email")
}

// SetEmail sets the given email in the git configuration
func (g *GitCLI) SetEmail(dir string, email string) error {
	// Will return status 1 silently if the email is not set.
	_, err := g.gitCmdWithOutput(dir, "config", "--global", "--get", "user.email")
	if err != nil {
		return g.gitCmd(dir, "config", "--global", "--add", "user.email", email)
	}
	return nil
}

// CreateBranch creates a branch with the given name in the Git repository from the given directory
func (g *GitCLI) CreateBranch(dir string, branch string) error {
	return g.gitCmd(dir, "branch", branch)
}

// Diff runs git diff
func (g *GitCLI) Diff(dir string) (string, error) {
	return g.gitCmdWithOutput(dir, "diff")
}

// FetchUnshallow runs git fetch --unshallow in dir
func (g *GitCLI) FetchUnshallow(dir string) error {
	err := g.gitCmd(dir, "fetch", "--unshallow")
	if err != nil {
		return errors.Wrapf(err, "running git fetch --unshallow %s", dir)
	}
	return nil
}

// IsShallow runs git rev-parse --is-shallow-repository in dir and returns the result
func (g *GitCLI) IsShallow(dir string) (bool, error) {
	out, err := g.gitCmdWithOutput(dir, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return false, errors.Wrapf(err, "running git rev-parse --is-shallow-repository %s", dir)
	}
	if out == "--is-shallow-repository" {
		// Newer git has a method to do it, but we use an old git in our builders, so resort to doing it manually
		gitDir, _, err := g.FindGitConfigDir(dir)
		if err != nil {
			return false, errors.Wrapf(err, "finding .git for %s", dir)
		}
		if _, err := os.Stat(filepath.Join(gitDir, ".git", "shallow")); os.IsNotExist(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return true, nil
	}
	b, err := util.ParseBool(out)
	if err != nil {
		return false, errors.Wrapf(err, "converting %v to bool", b)
	}
	return b, nil

}

// CreateBranchFrom creates a new branch called branchName from startPoint
func (g *GitCLI) CreateBranchFrom(dir string, branchName string, startPoint string) error {
	return g.gitCmd(dir, "branch", branchName, startPoint)
}
