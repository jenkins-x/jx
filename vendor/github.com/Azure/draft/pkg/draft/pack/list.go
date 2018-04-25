package pack

import (
	"github.com/Azure/draft/pkg/draft/pack/repo"
)

// List returns a list of all pack names found in the specified repository or error.
//
// If repoName == "", List returns the set of all packs aggregated across all repositories.
func List(packsDir, repoName string) ([]string, error) {
	var packs []string
	for _, r := range repo.FindRepositories(packsDir) {
		if repoName != "" && repoName != r.Name {
			continue
		}
		all, err := r.List()
		if err != nil {
			return packs, err
		}
		packs = append(packs, all...)
	}
	return packs, nil
}
