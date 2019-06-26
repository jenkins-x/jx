package releases_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/gits/releases"
	"github.com/stretchr/testify/assert"
)

func TestParseFullDependencyUpdateMessage(t *testing.T) {
	msg := `chore(dependencies): update https://github.com/pmuir/brie from 1.2.3 to 1.2.4

update BRIE_VERSION to 1.2.4`

	update, err := releases.ParseDependencyUpdateMessage(msg)
	assert.NoError(t, err)
	assert.Equal(t, "pmuir", update.Owner)
	assert.Equal(t, "brie", update.Repo)
	assert.Equal(t, "1.2.4", update.ToVersion)
	assert.Equal(t, "1.2.3", update.FromVersion)
	assert.Equal(t, "github.com", update.Host)
}

func TestParseSimpleDependencyUpdateMessage(t *testing.T) {
	msg := `chore(dependencies): update pmuir/brie from 1.2.3 to 1.2.4

update BRIE_VERSION to 1.2.4`

	update, err := releases.ParseDependencyUpdateMessage(msg)
	assert.NoError(t, err)
	assert.Equal(t, "pmuir", update.Owner)
	assert.Equal(t, "brie", update.Repo)
	assert.Equal(t, "1.2.4", update.ToVersion)
	assert.Equal(t, "1.2.3", update.FromVersion)
	assert.Equal(t, "", update.Host)
}
