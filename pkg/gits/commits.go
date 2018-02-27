package gits

import (
	"bytes"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/util"
	"strings"
)

type CommitInfo struct {
	Kind    string
	Feature string
	Message string
	group   *CommitGroup
}

type CommitGroup struct {
	Title string
	Order int
}

var (
	otherKindOrder = 10

	groupCounter = 0

	// ConventionalCommitTitles textual descriptions for
	// Conventional Commit types: https://conventionalcommits.org/
	ConventionalCommitTitles = map[string]*CommitGroup{
		"feat":     createCommitGroup("Features"),
		"fix":      createCommitGroup("Bug Fixes"),
		"perf":     createCommitGroup("Performance Improvements"),
		"refactor": createCommitGroup("Code Refactoring"),
		"docs":     createCommitGroup("Documentation"),
		"test":     createCommitGroup("Tests"),
		"revert":   createCommitGroup("Reverts"),
		"style":    createCommitGroup("Styles"),
		"chore":    createCommitGroup("Chores"),
		"":         createCommitGroup(""),
	}
)

func createCommitGroup(title string) *CommitGroup {
	groupCounter += 1
	return &CommitGroup{
		Title: title,
		Order: groupCounter,
	}
}

// ConventionalCommitTypeToTitle returns the title of the conventional commit type
// see: https://conventionalcommits.org/
func ConventionalCommitTypeToTitle(kind string) *CommitGroup {
	answer := ConventionalCommitTitles[strings.ToLower(kind)]
	if answer == nil {
		answer = &CommitGroup{strings.Title(kind), otherKindOrder}
	}
	return answer
}

// ParseCommit parses a conventional commit
// see: https://conventionalcommits.org/
func ParseCommit(message string) *CommitInfo {
	answer := &CommitInfo{
		Message: message,
	}

	idx := strings.Index(message, ":")
	if idx > 0 {
		answer.Kind = message[0:idx]

		rest := message[idx+1:]
		if strings.HasPrefix(rest, "(") {
			idx = strings.Index(rest, ")")
			if idx > 0 {
				answer.Feature = rest[1:idx]
				rest = strings.TrimSpace(rest[idx+1:])
			}
		}
		answer.Message = rest
	}
	return answer
}

func (c *CommitInfo) Group() *CommitGroup {
	if c.group == nil {
		c.group = ConventionalCommitTitles[strings.ToLower(c.Kind)]
	}
	return c.group
}

func (c *CommitInfo) Title() string {
	return c.Group().Title
}

func (c *CommitInfo) Order() int {
	return c.Group().Order
}

type GroupAndCommitInfos struct {
	group   *CommitGroup
	commits []string
}

// GenerateMarkdown generates the markdown document for the commits
func GenerateMarkdown(releaseSpec *v1.ReleaseSpec, gitInfo *GitRepositoryInfo) (string, error) {
	commitInfos := []*CommitInfo{}

	groupAndCommits := map[int]*GroupAndCommitInfos{}

	for _, cs := range releaseSpec.Commits {
		message := cs.Message
		if message != "" {
			ci := ParseCommit(message)

			description := "* " + describeCommit(&cs, ci) + "\n"
			group := ci.Group()
			if group != nil {
				gac := groupAndCommits[group.Order]
				if gac == nil {
					gac = &GroupAndCommitInfos{
						group:   group,
						commits: []string{},
					}
					groupAndCommits[group.Order] = gac
				}
				gac.commits = append(gac.commits, description)
			}
			commitInfos = append(commitInfos, ci)
		}
	}

	issues := releaseSpec.Issues
	prs := releaseSpec.PullRequests

	var buffer bytes.Buffer
	if len(commitInfos) == 0 && len(issues) == 0 && len(prs) == 0 {
		return "", nil
	}

	buffer.WriteString("## Changes\n\n")

	if len(issues) > 0 {
		buffer.WriteString("\n### Issues\n\n")

		for _, issue := range issues {
			buffer.WriteString("* " + describeIssue(gitInfo, &issue) + "\n")
		}
	}
	if len(prs) > 0 {
		buffer.WriteString("\n### Pull Requests\n\n")

		for _, pr := range prs {
			buffer.WriteString("* " + describeIssue(gitInfo, &pr) + "\n")
		}
	}

	for i := 0; i < groupCounter; i++ {
		gac := groupAndCommits[i]
		if gac != nil && len(gac.commits) > 0 {
			group := gac.group
			if group != nil && group.Title != "" {
				buffer.WriteString("\n### " + group.Title + "\n\n")
			}
			for _, msg := range gac.commits {
				buffer.WriteString(msg)
			}
		}
	}
	return buffer.String(), nil
}

func describeIssue(info *GitRepositoryInfo, issue *v1.IssueSummary) string {
	postfix := ""
	user := issue.User
	if user != nil {
		userText := ""
		login := user.Login
		url := user.URL
		label := login
		if label == "" {
			label = user.Name
		}
		if url == "" && login != "" {
			url = util.UrlJoin(info.HostURL(), login)
		}
		if url == "" {
			userText = label
		} else {
			if label != "" {
				userText = "[" + label + "](" + url + ")"
			}
		}
		if userText != "" {
			postfix = " (" + userText + ")"
		}
	}
	return "[#" + issue.ID + "](" + issue.URL + ") " + issue.Title + postfix
}

func describeCommit(cs *v1.CommitSummary, ci *CommitInfo) string {
	prefix := ""
	if ci.Feature != "" {
		prefix = ci.Feature + ":"
	}
	message := strings.TrimSpace(ci.Message)
	lines := strings.Split(message, "\n")

	// TODO add link to issue etc...
	return prefix + lines[0]
}
