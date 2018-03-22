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

package tos

import (
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/ttesting"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func createTosTestStructure(folder string, expect ttesting.Expect) {
	err := os.RemoveAll("/tmp/tgo_tos")
	if !os.IsNotExist(err) {
		expect.NoError(err)
	}

	baseFolder := "/tmp/tgo_tos/" + folder

	expect.NoError(os.MkdirAll(baseFolder+"/test1/test2a", 0777))
	expect.NoError(os.MkdirAll(baseFolder+"/test1/test2b", 0777))
	expect.NoError(os.MkdirAll(baseFolder+"/test3", 0777))

	file, err := os.Create(baseFolder + "/1.test")
	expect.NoError(err)
	_, err = file.WriteString("test1")
	expect.NoError(err)
	expect.NoError(file.Close())

	file, err = os.Create(baseFolder + "/test1/2.test")
	expect.NoError(err)
	_, err = file.WriteString("test2")
	expect.NoError(err)
	expect.NoError(file.Close())

	file, err = os.Create(baseFolder + "/test1/test2a/3.test")
	expect.NoError(err)
	_, err = file.WriteString("test3")
	expect.NoError(err)
	expect.NoError(file.Close())

	err = os.Symlink("1.test", baseFolder+"/test1/alias")
	if !os.IsExist(err) {
		expect.NoError(err)
	}
}

func TestCopy(t *testing.T) {
	expect := ttesting.NewExpect(t)
	createTosTestStructure("copy", expect)

	expect.NoError(Copy("/tmp/tgo_tos/copy_target", "/tmp/tgo_tos/copy"))

	expect.True(tio.FileExists("/tmp/tgo_tos/copy_target/1.test"))
	expect.True(tio.FileExists("/tmp/tgo_tos/copy_target/test1"))
	expect.True(tio.FileExists("/tmp/tgo_tos/copy_target/test1/2.test"))
	expect.True(tio.FileExists("/tmp/tgo_tos/copy_target/test1/test2a"))
	expect.True(tio.FileExists("/tmp/tgo_tos/copy_target/test1/test2a/3.test"))
	expect.True(tio.FileExists("/tmp/tgo_tos/copy_target/test1/test2b"))
	expect.True(tio.FileExists("/tmp/tgo_tos/copy_target/test3"))

	file, err := os.Open("/tmp/tgo_tos/copy_target/1.test")
	content := make([]byte, 5) // !! Assumes content is "testX"
	_, err = io.ReadFull(file, content)
	expect.NoError(err)
	expect.Equal("test1", string(content))
	expect.NoError(file.Close())

	file, err = os.Open("/tmp/tgo_tos/copy_target/test1/2.test")
	_, err = io.ReadFull(file, content)
	expect.NoError(err)
	expect.Equal("test2", string(content))
	expect.NoError(file.Close())

	file, err = os.Open("/tmp/tgo_tos/copy_target/test1/test2a/3.test")
	_, err = io.ReadFull(file, content)
	expect.NoError(err)
	expect.Equal("test3", string(content))
	expect.NoError(file.Close())

	link, err := os.Readlink("/tmp/tgo_tos/copy_target/test1/alias")
	expect.NoError(err)
	expect.Equal("1.test", link)

	expect.NoError(os.RemoveAll("/tmp/tgo_tos"))
}

func TestChmod(t *testing.T) {
	expect := ttesting.NewExpect(t)
	createTosTestStructure("chmod", expect)
	setMode := os.FileMode(0777)

	expect.NoError(Chmod("/tmp/tgo_tos/chmod", setMode))

	filepath.Walk("/tmp/tgo_tos/chmod", func(path string, info os.FileInfo, err error) error {
		// TODO: os.Chmod fails on symlinks
		if info.Mode()&os.ModeSymlink == 0 {
			expect.Equal(setMode, info.Mode()&0777)
		}
		return err
	})

	expect.NoError(os.RemoveAll("/tmp/tgo_tos"))
}

// Removed as not available on all platforms, requires root, etc.
/*
func TestChown(t *testing.T) {
	expect := ttesting.NewExpect(t)
	currentUser, err := user.Current()
	expect.NoError(err)

	if currentUser.Username != "root" {
		return // ### return, only root can chown without restrictions ###
	}

	createTosTestStructure("chown", expect)

	expect.NoError(Chown("/tmp/tgo_tos/chown", NobodyUid, NobodyGid))

	filepath.Walk("/tmp/tgo_tos/chown", func(path string, info os.FileInfo, err error) error {
		// TODO: os.Chmod fails on symlinks
		if info.Mode()&os.ModeSymlink == 0 {
			usr, grp, err := GetFileCredentialsName(path)
			expect.NoError(err)
			expect.Equal("nobody", usr)
			expect.Equal("nobody", grp)
		}
		return err
	})

	expect.NoError(os.RemoveAll("/tmp/tgo_tos"))
}
*/
