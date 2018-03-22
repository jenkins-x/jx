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
	"os/user"
	"strconv"
)

// GetUid returns the user id for a given user name
func GetUid(name string) (int, error) {
	switch name {
	case "nobody":
		return NobodyUid, nil

	case "root":
		return RootUid, nil

	default:
		userInfo, err := user.Lookup(name)
		if err != nil {
			return 0, err
		}

		return strconv.Atoi(userInfo.Uid)
	}
}

// GetGid returns the group id for a given group name
func GetGid(name string) (int, error) {
	switch name {
	case "nobody":
		return NobodyGid, nil

	case "root":
		return RootGid, nil

	default:
		groupInfo, err := user.LookupGroup(name)
		if err != nil {
			return 0, err
		}

		return strconv.Atoi(groupInfo.Gid)
	}
}
