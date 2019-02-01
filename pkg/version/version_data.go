package version

import (
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

type VersionKind string

var (
	// KindChart represents a chart version
	KindChart VersionKind = "charts"

	// KindPackage represents a package version
	KindPackage VersionKind = "packages"
)
// VersionData stores the information about 
type VersionData struct {
	Version string
	GitURL  string
	URL     string
}

// LoadVersionData loads a version data from the version configuration directory returning an empty object if there is
// no specific configuration available
func LoadVersionData(wrkDir string, kind VersionKind, name string) (*VersionData, error) {
	version := &VersionData{}

	path := filepath.Join(wrkDir, string(kind), name+".yml")

	exists, err := util.FileExists(path)
	if err != nil {
		return version, errors.Wrapf(err, "failed to cehck if file exists %s", path)
	}
	if !exists {
		return version, nil
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return version, errors.Wrapf(err, "failed to load YAML file %s", path)
	}
	err = yaml.Unmarshal(data, version)
	if err != nil {
		return version, errors.Wrapf(err, "failed to unmarshal YAML for file %s", path)
	}

	return version, err
}

// LoadVersionNumber loads just the version number for the given kind and name
func LoadVersionNumber(wrkDir string, kind VersionKind, name string) (string, error) {
	data, err := LoadVersionData(wrkDir, kind, name)
	if err != nil {
	  return "", err
	}
	return data.Version, err
}

