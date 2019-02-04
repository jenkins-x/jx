package version

import (
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

// VersionKind represents the kind of version
type VersionKind string

var (
	// KindChart represents a chart version
	KindChart VersionKind = "charts"

	// KindPackage represents a package version
	KindPackage VersionKind = "packages"
)

// StableVersion stores the stable version information
type StableVersion struct {
	Version string
	GitURL  string
	URL     string
}

// LoadStableVersion loads the stable version data from the version configuration directory returning an empty object if there is
// no specific stable version configuration available
func LoadStableVersion(wrkDir string, kind VersionKind, name string) (*StableVersion, error) {
	version := &StableVersion{}

	path := filepath.Join(wrkDir, string(kind), name+".yml")

	exists, err := util.FileExists(path)
	if err != nil {
		return version, errors.Wrapf(err, "failed to check if file exists %s", path)
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

// LoadStableVersionNumber loads just the stable version number for the given kind and name
func LoadStableVersionNumber(wrkDir string, kind VersionKind, name string) (string, error) {
	data, err := LoadStableVersion(wrkDir, kind, name)
	if err != nil {
		return "", err
	}
	version := data.Version
	if version != "" {
		log.Infof("using stable version %s from %s of %s from %s\n", util.ColorInfo(version), string(kind), util.ColorInfo(name), wrkDir)
	} else {
		log.Warnf("could not find a stable version from %s of %s from %s - we should lock down this chart version to improve stability. See: https://github.com/jenkins-x/jenkins-x-versions/blob/master/README.md\n", string(kind), name, wrkDir)
	}
	return version, err
}
