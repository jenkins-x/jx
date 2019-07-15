package version

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Callback a callback function for processing version information. Return true to continue processing
// or false to terminate the loop
type Callback func(kind VersionKind, name string, version *StableVersion) (bool, error)

// VersionKind represents the kind of version
type VersionKind string

const (
	// KindChart represents a chart version
	KindChart VersionKind = "charts"

	// KindPackage represents a package version
	KindPackage VersionKind = "packages"

	// KindDocker represents a docker resolveImage version
	KindDocker VersionKind = "docker"
)

var (
	// Kinds all the version kinds
	Kinds = []VersionKind{
		KindChart,
		KindPackage,
		KindDocker,
	}
	// KindStrings all the kinds as strings for validating CLI arguments
	KindStrings = []string{
		string(KindChart),
		string(KindPackage),
		string(KindDocker),
	}
)

// StableVersion stores the stable version information
type StableVersion struct {
	// Version the default version to use
	Version string `json:"version,omitempty"`
	// VersionUpperLimit represents the upper limit which indicates a version which is too new.

	// e.g. for packages we could use: `{ version: "1.10.1", upperLimit: "1.14.0"}` which would mean these
	// versions are all valid `["1.11.5", "1.13.1234"]` but these are invalid `["1.14.0", "1.14.1"]`
	UpperLimit string `json:"upperLimit,omitempty"`
	// GitURL the URL to the source code
	GitURL string `json:"gitUrl,omitempty"`
	// URL the URL for the documentation
	URL string `json:"url,omitempty"`
}

// VerifyPackage verifies the current version of the package is valid
func (data *StableVersion) VerifyPackage(name string, currentVersion string, workDir string) error {
	currentVersion = convertToVersion(currentVersion)
	if currentVersion == "" {
		return nil
	}
	version := convertToVersion(data.Version)
	if version == "" {
		log.Logger().Warnf("could not find a stable package version for %s from %s\nFor background see: https://jenkins-x.io/architecture/version-stream/", name, workDir)
		log.Logger().Infof("Please lock this version down via the command: %s", util.ColorInfo(fmt.Sprintf("jx step create version pr -k package -n %s", name)))
		return nil
	}

	currentSem, err := semver.Make(currentVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to parse semantic version for current version %s for package %s", currentVersion, name)
	}

	minSem, err := semver.Make(version)
	if err != nil {
		return errors.Wrapf(err, "failed to parse required semantic version %s for package %s", version, name)
	}

	upperLimitText := convertToVersion(data.UpperLimit)
	if upperLimitText == "" {
		if minSem.Equals(currentSem) {
			return nil
		}
		return verifyError(name, fmt.Errorf("package %s is on version %s but the version stream requires version %s", name, currentVersion, version))
	}

	// lets make sure the current version is in the range
	if currentSem.LT(minSem) {
		return verifyError(name, fmt.Errorf("package %s is an old version %s. The version stream requires at least %s", name, currentVersion, version))
	}

	limitSem, err := semver.Make(upperLimitText)
	if err != nil {
		return errors.Wrapf(err, "failed to parse upper limit version %s for package %s", upperLimitText, name)
	}

	if currentSem.GE(limitSem) {
		return verifyError(name, fmt.Errorf("package %s is using version %s which is too new. The version stream requires a version earlier than %s", name, currentVersion, upperLimitText))
	}
	return nil
}

// verifyError allows package verify errors to be disabled in development via environment variables
func verifyError(name string, err error) error {
	envVar := "JX_DISABLE_VERIFY_" + strings.ToUpper(name)
	value := os.Getenv(envVar)
	if strings.ToLower(value) == "true" {
		log.Logger().Warnf("$%s is true so disabling verify of %s: %s\n", envVar, name, err.Error())
		return nil
	}
	return err
}

// removes any whitespace and `v` prefix from a version string
func convertToVersion(text string) string {
	answer := strings.TrimSpace(text)
	answer = strings.TrimPrefix(answer, "v")
	return answer
}

// LoadStableVersion loads the stable version data from the version configuration directory returning an empty object if there is
// no specific stable version configuration available
func LoadStableVersion(wrkDir string, kind VersionKind, name string) (*StableVersion, error) {
	path := filepath.Join(wrkDir, string(kind), name+".yml")
	return LoadStableVersionFile(path)
}

// LoadStableVersionFile loads the stable version data from the given file name
func LoadStableVersionFile(path string) (*StableVersion, error) {
	version := &StableVersion{}
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
	version, err = LoadStableVersionFromData(data)
	if err != nil {
		return version, errors.Wrapf(err, "failed to unmarshal YAML for file %s", path)
	}
	return version, err
}

// LoadStableVersionFromData loads the stable version data from the given the data
func LoadStableVersionFromData(data []byte) (*StableVersion, error) {
	version := &StableVersion{}
	err := yaml.Unmarshal(data, version)
	if err != nil {
		return version, errors.Wrapf(err, "failed to unmarshal YAML")
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
		log.Logger().Debugf("using stable version %s from %s of %s from %s", util.ColorInfo(version), string(kind), util.ColorInfo(name), wrkDir)
	} else {
		// lets not warn if building current dir chart
		if kind == KindChart && name == "." {
			return version, err
		}
		log.Logger().Warnf("could not find a stable version from %s of %s from %s\nFor background see: https://jenkins-x.io/architecture/version-stream/", string(kind), name, wrkDir)
		log.Logger().Infof("Please lock this version down via the command: %s", util.ColorInfo(fmt.Sprintf("jx step create version pr -k %s -n %s", string(kind), name)))
	}
	return version, err
}

// SaveStableVersion saves the version file
func SaveStableVersion(wrkDir string, kind VersionKind, name string, stableVersion *StableVersion) error {
	path := filepath.Join(wrkDir, string(kind), name+".yml")
	return SaveStableVersionFile(path, stableVersion)
}

// SaveStableVersionFile saves the stabe version to the given file name
func SaveStableVersionFile(path string, stableVersion *StableVersion) error {
	data, err := yaml.Marshal(stableVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal data to YAML %#v", stableVersion)
	}
	dir, _ := filepath.Split(path)
	err = os.MkdirAll(dir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %s", dir)
	}

	err = ioutil.WriteFile(path, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", path)
	}
	return nil
}

// ForEachVersion processes all of the versions in the wrkDir using the given callback function.
func ForEachVersion(wrkDir string, callback Callback) error {
	for _, kind := range Kinds {
		err := ForEachKindVersion(wrkDir, kind, callback)
		if err != nil {
			return err
		}
	}
	return nil
}

// ForEachKindVersion processes all of the versions in the wrkDir and kind using the given callback function.
func ForEachKindVersion(wrkDir string, kind VersionKind, callback Callback) error {
	kindString := string(kind)
	kindDir := filepath.Join(wrkDir, kindString)
	glob := filepath.Join(kindDir, "*", "*.yml")
	paths, err := filepath.Glob(glob)
	if err != nil {
		return errors.Wrapf(err, "failed to find glob: %s", glob)
	}
	for _, path := range paths {
		versionData, err := LoadStableVersionFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load VersionData for file: %s", path)
		}
		name, err := filepath.Rel(kindDir, path)
		if err != nil {
			return errors.Wrapf(err, "failed to extract base path from %s", path)
		}
		ext := filepath.Ext(name)
		if ext != "" {
			name = strings.TrimSuffix(name, ext)
		}
		ok, err := callback(kind, name, versionData)
		if err != nil {
			return errors.Wrapf(err, "failed to process kind %s name %s", kindString, name)
		}
		if !ok {
			break
		}
	}
	return nil
}

// ResolveDockerImage resolves the version of the specified image against the version stream defined in versionsDir.
// If there is a version defined for the image in the version stream 'image:<version>' is returned, otherwise the
// passed image name is returned as is.
func ResolveDockerImage(versionsDir, image string) (string, error) {
	// lets check if we already have a version
	path := strings.SplitN(image, ":", 2)
	if len(path) == 2 && path[1] != "" {
		return image, nil
	}
	info, err := LoadStableVersion(versionsDir, KindDocker, image)
	if err != nil {
		return image, err
	}
	if info.Version == "" {
		// lets check if there is a docker.io prefix and if so lets try fetch without the docker prefix
		prefix := "docker.io/"
		if strings.HasPrefix(image, prefix) {
			image = strings.TrimPrefix(image, prefix)
			info, err = LoadStableVersion(versionsDir, KindDocker, image)
			if err != nil {
				return image, err
			}
		}
	}
	if info.Version == "" {
		log.Logger().Warnf("could not find a stable version for Docker image: %s in %s", image, versionsDir)
		log.Logger().Warn("for background see: https://jenkins-x.io/architecture/version-stream/")
		log.Logger().Infof("please lock this version down via the command: %s", util.ColorInfo(fmt.Sprintf("jx step create version pr -k docker -n %s -v 1.2.3", image)))
		return image, nil
	}
	prefix := strings.TrimSuffix(strings.TrimSpace(image), ":")
	return prefix + ":" + info.Version, nil
}
