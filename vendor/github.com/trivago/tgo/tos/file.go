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
	"io"
	"io/ioutil"
	"os"
)

// ChownByName is a wrapper around ChownId that allows changing user and group by name.
func ChownByName(filePath, usr, grp string) error {
	var uid, gid int
	var err error

	if uid, err = GetUid(usr); err != nil {
		return err
	}

	if gid, err = GetGid(grp); err != nil {
		return err
	}

	return Chown(filePath, uid, gid)
}

// ChownId is a wrapper around os.Chown that allows changing user and group
// recursively if given a directory.
func Chown(filePath string, uid, gid int) error {
	stat, err := os.Lstat(filePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			return err
		}
		for _, file := range files {
			if err := Chown(filePath+"/"+file.Name(), uid, gid); err != nil {
				return err
			}
		}
	}

	if stat.Mode()&os.ModeSymlink != 0 {
		// TODO: os.Chown fails on symlinks
		return nil
	}

	return os.Chown(filePath, uid, gid)
}

// Chmod is a wrapper around os.Chmod that allows changing rights recursively
// if a directory is given.
func Chmod(filePath string, mode os.FileMode) error {
	stat, err := os.Lstat(filePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			return err
		}
		for _, file := range files {
			if err := Chmod(filePath+"/"+file.Name(), mode); err != nil {
				return err
			}
		}

		// Set executable rights for folders if read or write is allowed
		execRights := 0
		if mode&0600 != 0 {
			execRights |= 0100
		}
		if mode&0060 != 0 {
			execRights |= 0010
		}
		if mode&0006 != 0 {
			execRights |= 0001
		}

		return os.Chmod(filePath, mode|os.FileMode(execRights))
	}

	if stat.Mode()&os.ModeSymlink != 0 {
		// TODO: os.Chmod fails on symlinks
		return nil
	}

	return os.Chmod(filePath, mode)
}

// IsSymlink returns true if a file is a symlink
func IsSymlink(file string) (bool, error) {
	fileStat, err := os.Lstat(file)
	if err != nil {
		return false, err
	}

	return fileStat.Mode()&os.ModeSymlink != 0, nil
}

// Copy is a file copy helper. Files will be copied to their destination,
// overwriting existing files. Already existing files that are not part of the
// copy process will not be touched. If source is a directory it is walked
// recursively. Non-existing folders in dest will be created.
// Copy returns upon the first error encountered. In-between results will not
// be rolled back.
func Copy(dest, source string) error {
	srcStat, err := os.Lstat(source)
	if err != nil {
		return err
	}

	if srcStat.IsDir() {
		files, err := ioutil.ReadDir(source)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(dest, srcStat.Mode()); err != nil && !os.IsExist(err) {
			return err
		}

		for _, file := range files {
			if err := Copy(dest+"/"+file.Name(), source+"/"+file.Name()); err != nil {
				return err
			}
		}
		return nil // ### return, copy done ###
	}

	// Symlink handling
	if srcStat.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(source)
		if err != nil {
			return err
		}
		return os.Symlink(target, dest) // ### return, copy done ###
	}

	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return os.Chmod(dest, srcStat.Mode())
}
