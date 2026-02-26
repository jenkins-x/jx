// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/blang/semver"
	"github.com/jenkins-x/jx-logging/pkg/log"
)

// Build information. Populated at build-time.
var (
	Version      string
	Revision     string
	BuildDate    string
	GoVersion    string
	GitTreeState string
)

// Map provides the iterable version information.
var Map = map[string]string{
	"version":      Version,
	"revision":     Revision,
	"buildDate":    BuildDate,
	"goVersion":    GoVersion,
	"gitTreeState": GitTreeState,
}

const (
	// VersionPrefix string for setting pre-release etc
	VersionPrefix = ""

	// ExampleVersion shows an example version in the help
	// if no version could be found (which should never really happen!)
	ExampleVersion = "1.1.59"

	// TestVersion used in test cases for the current version if no
	// version can be found - such as if the version property is not properly
	// included in the go test flags.
	TestVersion = "2.0.404"

	// TestRevision can be used in tests if no revision is passed in the test flags
	TestRevision = "04b628f48"

	// TestTreeState can be used in tests if no tree state is passed in the test flags
	TestTreeState = "clean"

	// TestBuildDate can be used in tests if no build date is passed in the test flags
	TestBuildDate = "2020-05-31T14:51:38Z"

	// TestGoVersion can be used in tests if no version is passed in the test flags
	TestGoVersion = "1.13.8"
)

// GetVersion gets the current version string
func GetVersion() string {
	v := Map["version"]
	if v == "" {
		v = TestVersion
	}
	return v
}

// GetSemverVersion returns a semver.Version struct representing the current version
func GetSemverVersion() (semver.Version, error) {
	text := strings.TrimPrefix(GetVersion(), VersionPrefix)
	v, err := semver.Make(text)
	if err != nil {
		return v, errors.Wrapf(err, "failed to parse version %s", text)
	}
	return v, nil
}

// GetRevision returns the short SHA1 hashes given a given revision
func GetRevision() string {
	v := Map["revision"]
	if v == "" {
		v = TestRevision
	}
	return v
}

// GetTreeState returns the state of the working tree
func GetTreeState() string {
	v := Map["gitTreeState"]
	if v == "" {
		v = TestTreeState
	}
	return v
}

// GetBuildDate returns the build date for the binary
func GetBuildDate() string {
	v := Map["buildDate"]
	if v == "" {
		v = TestBuildDate
	}
	return v
}

// GetGoVersion returns the version of go used to build the binary
func GetGoVersion() string {
	v := Map["goVersion"]
	if v == "" {
		v = TestGoVersion
	}
	return v
}

// StringDefault returns the current version string or returns a dummy
// default value if there is an error
func StringDefault(defaultValue string) string {
	v, err := GetSemverVersion()
	if err == nil {
		return v.String()
	}
	log.Logger().Warnf("Warning failed to load version: %s", err)
	return defaultValue
}
