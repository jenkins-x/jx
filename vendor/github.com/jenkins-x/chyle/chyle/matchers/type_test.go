package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewType(t *testing.T) {
	assert.Equal(t, regularCommit{}, newType(regularType))
	assert.Equal(t, mergeCommit{}, newType(mergeType))
}
