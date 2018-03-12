package repo

import (
	"os"
	"path/filepath"
	"strings"
)

const PackDirName = "packs"

// Repository represents a pack repository.
type Repository struct {
	Name string
	Dir  string
}

// FindRepositories takes a given path and returns a list of repositories.
//
// Repositories are defined as directories with a "packs" directory present.
func FindRepositories(path string) []Repository {
	var repos []Repository
	// fail fast if directory does not exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return repos
	}
	filepath.Walk(path, func(walkPath string, f os.FileInfo, err error) error {
		// find all directories in walkPath that have a child directory called "packs"
		fileInfo, err := os.Stat(filepath.Join(walkPath, PackDirName))
		if err != nil {
			return nil
		}
		if fileInfo.IsDir() {
			repos = append(repos, Repository{
				Name: strings.TrimPrefix(walkPath, path+"/"),
				Dir:  walkPath,
			})
		}
		return nil
	})
	return repos
}
