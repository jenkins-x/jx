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

// +build !windows

package tio

import (
	"fmt"
	"os"
	"syscall"
)

// OpenNamedPipe opens or attaches to a named pipe given by name. If the pipe
// is created, perm will be used as file permissions.
// This function is not available on windows.
func OpenNamedPipe(name string, perm uint32) (*os.File, error) {
	if err := syscall.Mkfifo(name, perm); !os.IsExist(err) {
		return nil, fmt.Errorf("Failed to create named pipe %s: %s", name, err)
	}

	pipe, err := os.OpenFile(name, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("Could not open named pipe %s: %s", name, err)
	}

	stats, _ := pipe.Stat()
	if stats.Mode()&os.ModeNamedPipe == 0 {
		return nil, fmt.Errorf("File is not a named pipe %s", name)
	}

	return pipe, nil
}
