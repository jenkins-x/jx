package gits

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOwnerGitHubUser_isOwner(t *testing.T) {
	t.Parallel()
	isOwnerGitHubUser := IsOwnerGitHubUser("owner", "owner")
	assert.True(t, isOwnerGitHubUser, "The owner should be the same as the GitHubUser")
}

func TestIsOwnerGitHubUser_isNotOwner(t *testing.T) {
	t.Parallel()
	isOwnerGitHubUser := IsOwnerGitHubUser("owner", "notowner")
	assert.False(t, isOwnerGitHubUser, "The owner must not be the same as the GitHubUser")
}
