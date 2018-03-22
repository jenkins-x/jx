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
	"github.com/trivago/tgo/ttesting"
	"os"
	"sort"
	"testing"
	"time"
)

func TestListFilesByDateMatching(t *testing.T) {
	expect := ttesting.NewExpect(t)

	files, err := ListFilesByDateMatching(".", "\\.go$")
	expect.NoError(err)
	expect.Greater(len(files), 1)

	lastTime := int64(0)
	diffCount := 0

	for _, file := range files {
		if expect.Leq(lastTime, file.ModTime().UnixNano()) {
			if lastTime != file.ModTime().UnixNano() {
				diffCount++
			}
		}
		lastTime = file.ModTime().UnixNano()
	}

	expect.Greater(diffCount, 0)
}

type fileInfoMock struct {
	os.FileInfo
	name string
	mod  time.Time
}

func (info fileInfoMock) Name() string {
	return info.name
}

func (info fileInfoMock) Size() int64 {
	return 0
}

func (info fileInfoMock) Mode() os.FileMode {
	return os.FileMode(0)
}

func (info fileInfoMock) ModTime() time.Time {
	return info.mod
}

func (info fileInfoMock) IsDir() bool {
	return false
}

func (info fileInfoMock) Sys() interface{} {
	return nil
}

func TestSplitPath(t *testing.T) {
	expect := ttesting.NewExpect(t)

	dir, name, ext := SplitPath("a/b")

	expect.Equal("a", dir)
	expect.Equal("b", name)
	expect.Equal("", ext)

	dir, name, ext = SplitPath("a/b.c")

	expect.Equal("a", dir)
	expect.Equal("b", name)
	expect.Equal(".c", ext)

	dir, name, ext = SplitPath("b")

	expect.Equal(".", dir)
	expect.Equal("b", name)
	expect.Equal("", ext)
}

func TestFilesByDate(t *testing.T) {
	expect := ttesting.NewExpect(t)

	testData := FilesByDate{
		fileInfoMock{name: "log1", mod: time.Unix(1, 0)},
		fileInfoMock{name: "log3", mod: time.Unix(0, 0)},
		fileInfoMock{name: "log2", mod: time.Unix(0, 0)},
		fileInfoMock{name: "log4", mod: time.Unix(2, 0)},
		fileInfoMock{name: "alog5", mod: time.Unix(0, 0)},
	}

	sort.Sort(testData)

	expect.Equal("alog5", testData[0].Name())
	expect.Equal("log2", testData[1].Name())
	expect.Equal("log3", testData[2].Name())
	expect.Equal("log1", testData[3].Name())
	expect.Equal("log4", testData[4].Name())
}

func TestFileExists(t *testing.T) {
	expect := ttesting.NewExpect(t)
	expect.True(FileExists("."))
	expect.False(FileExists("__foo.bar"))
}

func TestCommonPath(t *testing.T) {
	expect := ttesting.NewExpect(t)

	common := CommonPath("a", "a")
	expect.Equal("a", common)

	common = CommonPath("a", "b")
	expect.Equal("", common)

	common = CommonPath("a", "ab")
	expect.Equal("", common)

	common = CommonPath("a/b", "a")
	expect.Equal("a", common)

	common = CommonPath("b/a", "a")
	expect.Equal("", common)

	common = CommonPath("a/b", "a/b")
	expect.Equal("a/b", common)

	common = CommonPath("a/b", "a/bc")
	expect.Equal("a", common)

	common = CommonPath("a/b/c", "a/b/d")
	expect.Equal("a/b", common)
}
