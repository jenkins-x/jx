package gits

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func testParseCommits(t *testing.T) {
	assertParseCommit(t, "something regular", CommitInfo{
		Message: "something regular",
	})
	assertParseCommit(t, "feat: cheese", CommitInfo{
		Kind:    "feat",
		Message: "cheese",
	})
	assertParseCommit(t, "feat(beer): wine is good too", CommitInfo{
		Kind:    "feat",
		Feature: "beer",
		Message: "wine is good too",
	})
}

func assertParseCommit(t *testing.T, input string, expected CommitInfo) {
	info := ParseCommit(input)
	assert.NotNil(t, info)
	assert.Equal(t, expected.Kind, info.Kind, "Kind for Commit %s", info)
	assert.Equal(t, expected.Feature, info.Feature, "Feature for Commit %s", info)
	assert.Equal(t, expected.Message, info.Message, "Message for Commit %s", info)
	assert.Equal(t, expected, info, "CommitInfo for Commit %s", info)
}
