package gits

import (
	"bytes"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/util"
	"strconv"
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
	groupCounter = 0

	// ConventionalCommitTitles textual descriptions for
	// Conventional Commit types: https://conventionalcommits.org/
	ConventionalCommitTitles = map[string]*CommitGroup{
		"feat":     createCommitGroup("New Features"),
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

	unknownKindOrder = groupCounter + 1
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
		answer = &CommitGroup{strings.Title(kind), unknownKindOrder}
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

		rest := strings.TrimSpace(message[idx+1:])
		if strings.HasPrefix(rest, "(") {
			idx = strings.Index(rest, ")")
			if idx > 0 {
				answer.Feature = strings.TrimSpace(rest[1:idx])
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
func GenerateMarkdown(releaseSpec *v1.ReleaseSpec, gitInfo *GitRepository) (string, error) {
	commitInfos := []*CommitInfo{}

	groupAndCommits := map[int]*GroupAndCommitInfos{}

	issues := releaseSpec.Issues
	issueMap := map[string]*v1.IssueSummary{}
	for _, issue := range issues {
		copy := issue
		issueMap[copy.ID] = &copy
	}

	for _, cs := range releaseSpec.Commits {
		message := cs.Message
		if message != "" {
			ci := ParseCommit(message)

			description := "* " + describeCommit(gitInfo, &cs, ci, issueMap) + "\n"
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

	prs := releaseSpec.PullRequests

	var buffer bytes.Buffer
	if len(commitInfos) == 0 && len(issues) == 0 && len(prs) == 0 {
		return "", nil
	}

	buffer.WriteString("## Changes\n")

	hasTitle := false
	for i := 0; i <= unknownKindOrder; i++ {
		gac := groupAndCommits[i]
		if gac != nil && len(gac.commits) > 0 {
			group := gac.group
			if group != nil {
				legend := ""
				buffer.WriteString("\n")
				if group.Title == "" && hasTitle {
					group.Title = "Other Changes"
					legend = "These commits did not use [Conventional Commits](https://conventionalcommits.org/) formatted messages:\n\n"
				}
				if group.Title != "" {
					hasTitle = true
					buffer.WriteString("### " + group.Title + "\n\n" + legend)
				}
			}
			previous := ""
			for _, msg := range gac.commits {
				if msg != previous {
					buffer.WriteString(msg)
					previous = msg
				}
			}
		}
	}

	if len(issues) > 0 {
		buffer.WriteString("\n### Issues\n\n")

		previous := ""
		for _, issue := range issues {
			msg := describeIssue(gitInfo, &issue)
			if msg != previous {
				buffer.WriteString("* " + msg + "\n")
				previous = msg
			}
		}
	}
	if len(prs) > 0 {
		buffer.WriteString("\n### Pull Requests\n\n")

		previous := ""
		for _, pr := range prs {
			msg := describeIssue(gitInfo, &pr)
			if msg != previous {
				buffer.WriteString("* " + msg + "\n")
				previous = msg
			}
		}
	}
	return buffer.String(), nil
}

func describeIssue(info *GitRepository, issue *v1.IssueSummary) string {
	return describeIssueShort(info, issue) + issue.Title + describeUser(info, issue.User)
}

func describeIssueShort(info *GitRepository, issue *v1.IssueSummary) string {
	prefix := ""
	id := issue.ID
	if len(id) > 0 {
		// lets only add the hash prefix for numeric ids
		_, err := strconv.Atoi(id)
		if err == nil {
			prefix = "#"
		}
	}
	return "[" + prefix + issue.ID + "](" + issue.URL + ") "
}

func describeUser(info *GitRepository, user *v1.UserDetails) string {
	answer := ""
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
			answer = " (" + userText + ")"
		}
	}
	return answer
}

func describeCommit(info *GitRepository, cs *v1.CommitSummary, ci *CommitInfo, issueMap map[string]*v1.IssueSummary) string {
	prefix := ""
	if ci.Feature != "" {
		prefix = ci.Feature + ": "
	}
	message := strings.TrimSpace(ci.Message)
	lines := strings.Split(message, "\n")

	// TODO add link to issue etc...
	user := cs.Author
	if user == nil {
		user = cs.Committer
	}
	issueText := ""
	for _, issueId := range cs.IssueIDs {
		issue := issueMap[issueId]
		if issue != nil {
			issueText += " " + describeIssueShort(info, issue)
		}
	}
	return prefix + lines[0] + describeUser(info, user) + issueText
}
