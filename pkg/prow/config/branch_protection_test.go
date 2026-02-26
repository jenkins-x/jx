// +build unit

package config

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config/branchprotection"
	"github.com/stretchr/testify/assert"
)

func newBranchProtection() branchprotection.Config {
	return branchprotection.Config{
		Orgs: map[string]branchprotection.Org{
			"org": {
				Repos: map[string]branchprotection.Repo{
					"repo": {
						Policy: branchprotection.Policy{},
						Branches: map[string]branchprotection.Branch{
							"master": {},
						}}}}},
	}
}

func TestRemoveRepoFromBranchProtection_HappyPath(t *testing.T) {
	t.Parallel()
	bp := newBranchProtection()

	assert.Contains(t, bp.Orgs["org"].Repos, "repo")

	err := RemoveRepoFromBranchProtection(&bp, "org/repo")

	assert.NoError(t, err)
	assert.NotContains(t, bp.Orgs["org"].Repos, "repo")
}

func TestRemoveRepoFromBranchProtection_NoOrg(t *testing.T) {
	t.Parallel()
	bp := newBranchProtection()

	err := RemoveRepoFromBranchProtection(&bp, "wrong-org/repo")

	assert.EqualError(t, err, "no repos found for org wrong-org")
	assert.Contains(t, bp.Orgs["org"].Repos, "repo")
}

func TestRemoveRepoFromBranchProtection_NoRepo(t *testing.T) {
	t.Parallel()
	bp := newBranchProtection()

	err := RemoveRepoFromBranchProtection(&bp, "org/wrong-repo")

	assert.EqualError(t, err, "repo wrong-repo not found in org org")
	assert.Contains(t, bp.Orgs["org"].Repos, "repo")
}
