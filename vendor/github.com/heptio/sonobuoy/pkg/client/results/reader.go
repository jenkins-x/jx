/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package results

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/pkg/errors"
)

const (
	// PluginsDir defines where in the archive the plugin results are.
	PluginsDir = "plugins/"

	hostsDir                  = "hosts/"
	namespacedResourcesDir    = "resources/ns/"
	nonNamespacedResourcesDir = "resources/cluster/"
	podLogs                   = "podlogs/"
	metadataDir               = "meta/"
	defaultServicesFile       = "Services.json"
	defaultNodesFile          = "Nodes.json"
	defaultServerVersionFile  = "serverversion.json"
	defaultServerGroupsFile   = "servergroups.json"
)

const (
	// UnknownVersion lets the consumer know if this client can detect the archive version or not.
	UnknownVersion = "v?.?"
	VersionEight   = "v0.8"
	VersionNine    = "v0.9"
	VersionTen     = "v0.10"
)

// Reader holds a reader and a version. It uses the version to know where to
// find files within the archive.
type Reader struct {
	io.Reader
	Version string
}

// NewReaderWithVersion creates a results.Reader that interprets a results
// archive of the version passed in.
// Useful if the reader can be read only once and if the version of the data to
// read is known.
func NewReaderWithVersion(reader io.Reader, version string) *Reader {
	return &Reader{
		Reader:  reader,
		Version: version,
	}
}

// NewReaderFromBytes is a helper constructor that will discover the version of the archive
// and return a new Reader with the correct version already populated.
func NewReaderFromBytes(data []byte) (*Reader, error) {
	r := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new gzip reader")
	}
	version, err := DiscoverVersion(gzipReader)
	if err != nil {
		return nil, errors.Wrap(err, "error discovering version")
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Wrap(err, "error seeking to start")
	}
	if err = gzipReader.Reset(r); err != nil {
		return nil, errors.Wrap(err, "error reseting gzip reader")
	}
	return &Reader{
		Reader:  gzipReader,
		Version: version,
	}, nil
}

// DiscoverVersion takes a Sonobuoy archive stream and extracts just the
// version of the archive.
func DiscoverVersion(reader io.Reader) (string, error) {
	r := &Reader{
		Reader: reader,
	}

	conf := &config.Config{}

	err := r.WalkFiles(func(path string, info os.FileInfo, err error) error {
		return ExtractConfig(path, info, conf)
	})
	if err != nil {
		return "", errors.Wrap(err, "error extracting config")
	}
	var version string
	// Get rid of any of the extra version information that doesn't affect archive layout.
	// Example: v0.10.0-a2b3d4
	if strings.HasPrefix(conf.Version, VersionEight) {
		version = VersionEight
	} else if strings.HasPrefix(conf.Version, VersionNine) {
		version = VersionNine
	} else if strings.HasPrefix(conf.Version, VersionTen) {
		version = VersionTen
	} else {
		return "", errors.New("cannot discover Sonobuoy archive version")
	}
	return version, nil
}

// tarFileInfo implements os.FileInfo and extends the Sys() method to
// return a reader to a file in a tar archive.
type tarFileInfo struct {
	os.FileInfo
	io.Reader
}

// Sys is going to be an io.Reader to a file in a tar archive.
// This is how data is extracted from the archive.
func (t *tarFileInfo) Sys() interface{} {
	return t.Reader
}

// WalkFiles walks all of the files in the archive.
func (r *Reader) WalkFiles(walkfn filepath.WalkFunc) error {
	tr := tar.NewReader(r)
	var err error
	var header *tar.Header
	for {
		header, err = tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error getting next file in archive")
		}
		info := &tarFileInfo{
			header.FileInfo(),
			tr,
		}
		err = walkfn(filepath.Clean(header.Name), info, err)
	}
	return nil
}

// Functions to be used within a walkfn.

// ExtractBytes pulls out bytes into a buffer for any path matching file.
func ExtractBytes(file string, path string, info os.FileInfo, buf *bytes.Buffer) error {
	if file == path {
		reader, ok := info.Sys().(io.Reader)
		if !ok {
			return errors.New("info.Sys() is not a reader")
		}
		_, err := buf.ReadFrom(reader)
		if err != nil {
			return errors.Wrap(err, "could not read from buffer")
		}
	}
	return nil
}

// ExtractIntoStruct takes a predicate function and some file information
// and decodes the contents of the file that matches the predicate into the
// interface passed in (generally a pointer to a struct/slice).
func ExtractIntoStruct(predicate func(string) bool, path string, info os.FileInfo, object interface{}) error {
	if predicate(path) {
		reader, ok := info.Sys().(io.Reader)
		if !ok {
			return errors.New("info.Sys() is not a reader")
		}
		// TODO(chuckha) Perhaps find a more robust way to handle different data formats.
		if strings.HasSuffix(path, "xml") {
			decoder := xml.NewDecoder(reader)
			if err := decoder.Decode(object); err != nil {
				return errors.Wrap(err, "error decoding xml into object")
			}
			return nil
		}

		// If it's not xml it's probably json
		decoder := json.NewDecoder(reader)
		if err := decoder.Decode(object); err != nil {
			return errors.Wrap(err, "error decoding json into object")
		}
	}
	return nil
}

// ExtractFileIntoStruct is a helper for a common use case of extracting
// the contents of one file into the object.
func ExtractFileIntoStruct(file, path string, info os.FileInfo, object interface{}) error {
	return ExtractIntoStruct(func(p string) bool {
		return file == p
	}, path, info, object)
}

// ExtractConfig populates the config object regardless of version.
func ExtractConfig(path string, info os.FileInfo, conf *config.Config) error {
	return ExtractIntoStruct(func(file string) bool {
		return path == ConfigFile(VersionTen) || path == ConfigFile(VersionEight)
	}, path, info, conf)
}

// Functions for helping with backwards compatibility

// Metadata is the location of the metadata directory in the results archive.
func (r *Reader) Metadata() string {
	return metadataDir
}

// ServerVersionFile is the location of the file that contains the Kubernetes
// version Sonobuoy ran on.
func (r *Reader) ServerVersionFile() string {
	switch r.Version {
	case VersionEight:
		return "serverversion/serverversion.json"
	default:
		return defaultServerVersionFile
	}
}

// NamespacedResources returns the path to the directory that contains
// information about namespaced Kubernetes resources.
func (r *Reader) NamespacedResources() string {
	return namespacedResourcesDir
}

// NonNamespacedResources returns the path to the non-namespaced directory.
func (r *Reader) NonNamespacedResources() string {
	switch r.Version {
	case VersionEight:
		return "resources/non-ns/"
	default:
		return nonNamespacedResourcesDir
	}
}

// NodesFile returns the path to the file that lists the nodes of the Kubernetes
// cluster.
func (r *Reader) NodesFile() string {
	return filepath.Join(r.NonNamespacedResources(), defaultNodesFile)
}

// ServerGroupsFile returns the path to the groups the Kubernetes API supported at the time of the run.
func (r *Reader) ServerGroupsFile() string {
	return defaultServerGroupsFile
}

// ConfigFile returns the path to the sonobuoy config file.
// This is not a method as it is used to determine the version of the archive.
func ConfigFile(version string) string {
	switch version {
	case VersionEight:
		return "config.json"
	default:
		return "meta/config.json"
	}
}
