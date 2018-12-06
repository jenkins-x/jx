package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetRemoteAndRepo(t *testing.T) {
	origin, repo, err := GetRemoteAndRepo("jenkins-x/jx")
	assert.Equal(t, origin, "jenkins-x")
	assert.Equal(t, repo, "jx")
	assert.NoError(t, err)

	origin, repo, err = GetRemoteAndRepo("too/many/slashes")
	assert.Error(t, err)

	origin, repo, err = GetRemoteAndRepo("not-enough-slashes")
	assert.Error(t, err)
}
