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

// +build !go1.7 !cgo

package tos

import (
	"fmt"
)

// GetFileCredentials returns the user and group id of a given path.
// This function is not supported on windows platforms.
func GetFileCredentials(name string) (uid int, gid int, err error) {
	return 0, 0, fmt.Errorf("Not supported on this platform")
}

// GetFileCredentialsName returns the user and group name of a given path.
// This function is not supported on windows platforms.
func GetFileCredentialsName(name string) (usr string, grp string, err error) {
	return "", "", fmt.Errorf("Not supported on this platform")
}
