package repo

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// ErrPackNotFoundInRepo is the error returned when a pack is not found in a pack repo
var ErrPackNotFoundInRepo = errors.New("pack not found in pack repo")

// PackDirName is name for the packs directory
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
				Name: filepath.ToSlash(strings.TrimPrefix(walkPath, path+string(os.PathSeparator))),
				Dir:  walkPath,
			})
		}
		return nil
	})
	return repos
}

// Pack finds a packs with the given name in a repository and returns path
func (r *Repository) Pack(name string) (string, error) {

	//confirm repo exists
	if _, err := os.Stat(r.Dir); os.IsNotExist(err) {
		return "", fmt.Errorf("pack repo %s not found", r.Name)
	}

	targetDir := filepath.Join(r.Dir, "packs", name)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return "", ErrPackNotFoundInRepo
	}

	return targetDir, nil
}

// List returns a slice of pack names in the repository or error.
//
// The returned pack names are prefixed by the repository name, e.g. "draft/go"
func (r *Repository) List() ([]string, error) {
	packsDir := filepath.Join(r.Dir, PackDirName)
	switch fi, err := os.Stat(packsDir); {
	case err != nil:
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("pack repo %s packs directory not found", r.Name)
		}
	case !fi.IsDir():
		return nil, fmt.Errorf("%s is not a directory", packsDir)
	}
	var packs []string
	files, err := ioutil.ReadDir(packsDir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			repoPack := filepath.ToSlash(filepath.Join(r.Name, file.Name()))
			packs = append(packs, repoPack)
		}
	}
	return packs, nil
}
