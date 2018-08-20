package gits

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseCommits(t *testing.T) {
	t.Parallel()
	assertParseCommit(t, "something regular", &CommitInfo{
		Message: "something regular",
	})
	assertParseCommit(t, "feat: cheese", &CommitInfo{
		Kind:    "feat",
		Message: "cheese",
	})
	assertParseCommit(t, "feat:(beer) wine is good too", &CommitInfo{
		Kind:    "feat",
		Feature: "beer",
		Message: "wine is good too",
	})
	assertParseCommit(t, "feat(beer): wine is good too", &CommitInfo{
		Kind:    "feat(beer)",
		Feature: "",
		Message: "wine is good too",
	})
}

func assertParseCommit(t *testing.T, input string, expected *CommitInfo) {
	info := ParseCommit(input)
	assert.NotNil(t, info)
	assert.Equal(t, expected.Kind, info.Kind, "Kind for Commit %s", info)
	assert.Equal(t, expected.Feature, info.Feature, "Feature for Commit %s", info)
	assert.Equal(t, expected.Message, info.Message, "Message for Commit %s", info)
	assert.Equal(t, expected, info, "CommitInfo for Commit %s", info)
}

func TestChangelogMarkdown(t *testing.T) {
	t.Parallel()
	releaseSpec := &v1.ReleaseSpec{
		Commits: []v1.CommitSummary{
			{
				Message: "some commit 1\nfixes #123",
				SHA:     "123",
				Author: &v1.UserDetails{
					Name:  "James Strachan",
					Login: "jstrachan",
				},
			},
			{
				Message: "some commit 2\nfixes #345",
				SHA:     "456",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
			},
		},
	}
	gitInfo := &GitRepositoryInfo{
		Host:         "github.com",
		Organisation: "jstrachan",
		Name:         "foo",
	}
	markdown, err := GenerateMarkdown(releaseSpec, gitInfo)
	assert.Nil(t, err)
	//t.Log("Generated => " + markdown)

	expectedMarkdown := `## Changes

* some commit 1 ([jstrachan](https://github.com/jstrachan))
* some commit 2 ([rawlingsj](https://github.com/rawlingsj))
`
	assert.Equal(t, expectedMarkdown, markdown)

}

func TestChangelogMarkdownWithConventionalCommits(t *testing.T) {
	t.Parallel()
	releaseSpec := &v1.ReleaseSpec{
		Commits: []v1.CommitSummary{
			{
				Message: "fix: some commit 1\nfixes #123",
				SHA:     "123",
				Author: &v1.UserDetails{
					Name:  "James Strachan",
					Login: "jstrachan",
				},
			},
			{
				Message: "feat: some commit 2\nfixes #345",
				SHA:     "456",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
			},
			{
				Message: "feat:(has actual feature name) some commit 3\nfixes #456",
				SHA:     "567",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
			},
			{
				Message: "bad comment 4",
				SHA:     "678",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
			},
		},
	}
	gitInfo := &GitRepositoryInfo{
		Host:         "github.com",
		Organisation: "jstrachan",
		Name:         "foo",
	}
	markdown, err := GenerateMarkdown(releaseSpec, gitInfo)
	assert.Nil(t, err)
	//t.Log("Generated => " + markdown)

	expectedMarkdown := `## Changes

### New Features

* some commit 2 ([rawlingsj](https://github.com/rawlingsj))
* has actual feature name: some commit 3 ([rawlingsj](https://github.com/rawlingsj))

### Bug Fixes

* some commit 1 ([jstrachan](https://github.com/jstrachan))

### Other Changes

These commits did not use [Conventional Commits](https://conventionalcommits.org/) formatted messages:

* bad comment 4 ([rawlingsj](https://github.com/rawlingsj))
`
	assert.Equal(t, expectedMarkdown, markdown)
}
