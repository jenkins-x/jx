// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tio

import (
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FilesByDate implements the Sort interface by Date for os.FileInfo arrays
type FilesByDate []os.FileInfo

// Len returns the number of files in the array
func (files FilesByDate) Len() int {
	return len(files)
}

// Swap exchanges the values stored at indexes a and b
func (files FilesByDate) Swap(a, b int) {
	files[a], files[b] = files[b], files[a]
}

// Less compares the date of the files stored at a and b as in
// "[a] modified < [b] modified". If both files were created in the same second
// the file names are compared by using a lexicographic string compare.
func (files FilesByDate) Less(a, b int) bool {
	timeA, timeB := files[a].ModTime().UnixNano(), files[b].ModTime().UnixNano()
	if timeA == timeB {
		return files[a].Name() < files[b].Name()
	}
	return timeA < timeB
}

// ListFilesByDateMatching gets all files from a directory that match a given
// regular expression pattern and orders them by modification date (ascending).
// Directories and symlinks are excluded from the returned list.
func ListFilesByDateMatching(directory string, pattern string) ([]os.FileInfo, error) {
	filteredFiles := []os.FileInfo{}
	filter, err := regexp.Compile(pattern)
	if err != nil {
		return filteredFiles, err
	}

	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return filteredFiles, err
	}

	sort.Sort(FilesByDate(files))

	for _, file := range files {
		if file.IsDir() || file.Mode()&os.ModeSymlink == os.ModeSymlink {
			continue // ### continue, skip symlinks and directories ###
		}
		if filter.MatchString(file.Name()) {
			filteredFiles = append(filteredFiles, file)
		}
	}

	return filteredFiles, nil
}

// SplitPath separates a file path into directory, filename (without extension)
// and file extension (with dot). If no directory could be derived "." is
// returned as a directory. If no file extension could be derived "" is returned
// as a file extension.
func SplitPath(filePath string) (dir string, base string, ext string) {
	dir = filepath.Dir(filePath)
	ext = filepath.Ext(filePath)
	base = filepath.Base(filePath)
	base = base[:len(base)-len(ext)]
	return dir, base, ext
}

// FileExists does a proper check on wether a file exists or not.
func FileExists(filePath string) bool {
	_, err := os.Lstat(filePath)
	return !os.IsNotExist(err)
}

// IsDirectory returns true if a given path points to a directory.
func IsDirectory(filePath string) bool {
	stat, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

// CommonPath returns the longest common path of both paths given.
func CommonPath(path1, path2 string) string {
	parts1 := strings.Split(path1, "/")
	parts2 := strings.Split(path2, "/")
	maxIdx := len(parts1)
	if len(parts2) < maxIdx {
		maxIdx = len(parts2)
	}

	common := make([]string, 0, maxIdx)
	for i := 0; i < maxIdx; i++ {
		if parts1[i] == parts2[i] {
			common = append(common, parts1[i])
		}
	}

	return strings.Join(common, "/")
}

// FileCRC32 returns the checksum of a given file
func FileCRC32(path string) (uint32, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return crc32.ChecksumIEEE(data), nil
}
