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

// +build cgo,go1.7

package tos

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// GetFileCredentials returns the user and group id of a given path.
// This function is not supported on windows platforms.
func GetFileCredentials(name string) (uid int, gid int, err error) {
	stat, err := os.Lstat(name)
	if err != nil {
		return 0, 0, err
	}

	nativeStat := stat.Sys().(*syscall.Stat_t)
	return int(nativeStat.Uid), int(nativeStat.Gid), nil
}

// GetFileCredentialsName returns the user and group name of a given path.
// This function is not supported on windows platforms.
func GetFileCredentialsName(name string) (usr string, grp string, err error) {
	uid, gid, err := GetFileCredentials(name)
	if err != nil {
		return "", "", err
	}

	usrData, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return "", "", err
	}

	// Requires go1.7
	grpData, err := user.LookupGroupId(strconv.Itoa(gid))
	if err != nil {
		return "", "", err
	}

	return usrData.Username, grpData.Name, nil
}
