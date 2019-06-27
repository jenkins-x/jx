package dependencymatrix

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/pkg/errors"
)

const (
	dependencyMatrixDirName = "dependency-matrix"
)

// DependencyMatrix is the dependency matrix for a git repo
type DependencyMatrix struct {
	Dependencies []Dependency `json:"dependencies"`
}

// Dependency represents a dependency in the DependencyMatrix
type Dependency struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	URL        string `json:"url"`
	Component  string `json:"component, omitempty"`
	Version    string `json:"version"`
	VersionURL string `json:"versionUrl"`
}

// UpdateDependencyMatrix updates the dependency matrix in dir/dependency-matrix using update
func UpdateDependencyMatrix(dir string, update *v1.DependencyUpdate) error {
	// Only create/update the dependency matrix if the directory already exists
	if info, err := os.Stat(filepath.Join(dir, dependencyMatrixDirName)); err == nil && info.IsDir() {
		var data []byte
		path := filepath.Join(dir, dependencyMatrixDirName, "matrix.yaml")
		info, err := os.Stat(path)
		if info.IsDir() {
			return errors.Errorf("%s is a directory", path)
		} else if err == nil {
			data, err = ioutil.ReadFile(path)
			if err != nil {
				return errors.Wrapf(err, "reading %s", path)
			}
		}
		var dependencyMatrix DependencyMatrix
		err = yaml.Unmarshal(data, &dependencyMatrix)
		if err != nil {
			return errors.Wrapf(err, "unmarshaling %s", path)
		}
		found := false
		for _, d := range dependencyMatrix.Dependencies {
			if d.Owner == update.Owner && d.Repo == update.Owner && d.Component == update.Component {
				d.Version = update.ToVersion
				found = true
				d.URL = update.URL
				d.Version = update.ToReleaseHTMLURL
			}
		}
		if !found {
			dependencyMatrix.Dependencies = append(dependencyMatrix.Dependencies, Dependency{
				Owner:      update.Owner,
				Repo:       update.Repo,
				Component:  update.Component,
				Version:    update.ToVersion,
				URL:        update.URL,
				VersionURL: update.ToReleaseHTMLURL,
			})
		}
		data, err = yaml.Marshal(dependencyMatrix)
		if err != nil {
			return errors.Wrapf(err, "marshaling %s", path)
		}
		err = ioutil.WriteFile(path, data, info.Mode())
		if err != nil {
			return errors.Wrapf(err, "writing %s", path)
		}
		err = GenerateMarkdownDependencyMatrix(filepath.Join(dir, dependencyMatrixDirName, "matrix.md"), dependencyMatrix)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

//GenerateMarkdownDependencyMatrix will generate a markdown version of the dependency matrix at path
func GenerateMarkdownDependencyMatrix(path string, matrix DependencyMatrix) error {
	var md bytes.Buffer
	md.WriteString("# Dependency Matrix\n")
	md.WriteString("\n")
	md.WriteString("Dependency | Component | Version\n")
	md.WriteString("---------- | --------- | -------\n")
	for _, d := range matrix.Dependencies {
		md.WriteString(fmt.Sprintf("[%s/%s](%s) | %s | [%s](%s)", d.Owner, d.Repo, d.URL, d.Component, d.Version, d.VersionURL))
	}
	err := ioutil.WriteFile(path, md.Bytes(), 0600)
	if err != nil {
		return errors.Wrapf(err, "writing %s", path)
	}
	return nil
}
