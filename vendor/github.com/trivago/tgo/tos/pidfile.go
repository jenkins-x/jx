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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
	"time"
)

const (
	InvalidPID = -1
)

// WritePidFile writes this a process id into a file.
// An error will be returned if the file already exists.
func WritePidFile(pid int, filename string) error {
	pidString := strconv.Itoa(pid)
	pidFile, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer pidFile.Close()
	_, err = pidFile.WriteString(pidString)
	return err
}

// WritePidFileForced writes this a process id into a file.
// An existing file will be overwritten.
func WritePidFileForced(pid int, filename string) error {
	pidString := strconv.Itoa(pid)
	return ioutil.WriteFile(filename, []byte(pidString), 0644)
}

// GetPidFromFile tries loads the content of a pid file.
// A pidfile is expected to contain only an integer with a valid process id.
func GetPidFromFile(filename string) (int, error) {
	var (
		pidBytes []byte
		pid      int
		err      error
	)

	if pidBytes, err = ioutil.ReadFile(filename); err != nil {
		return InvalidPID, fmt.Errorf("Could not read pidfile %s: %s", filename, err)
	}

	pidString := string(bytes.Trim(pidBytes, "\r\n\t "))
	if pid, err = strconv.Atoi(string(pidString)); err != nil {
		return InvalidPID, fmt.Errorf("Could not read pid from pidfile %s: %s", filename, err)
	}

	return pid, nil
}

// GetProcFromFile utilizes GetPidFromFile to create a os.Process handle for
// the pid contained in the given pid file.
func GetProcFromFile(filename string) (*os.Process, error) {
	var (
		pid  int
		err  error
		proc *os.Process
	)

	if pid, err = GetPidFromFile(filename); err != nil {
		return nil, err
	}

	// FindProcess always returns a proc on unix
	if proc, err = os.FindProcess(pid); err != nil {
		return nil, err
	}

	// Try to signal the process to check if it is running
	if err = proc.Signal(syscall.Signal(0)); err != nil {
		return nil, err
	}

	return proc, nil
}

// Terminate tries to gracefully shutdown a process by sending SIGTERM.
// If the process does not shut down after (at least) gracePeriod a SIGKILL will be sent.
// 10 checks will be done during gracePeriod to check if the process is still
// alive. If a gracePeriod of 0 is passed SIGKILL will be send immediately.
func Terminate(proc *os.Process, gracePeriod time.Duration) error {
	if gracePeriod > 0 {
		err := proc.Signal(syscall.SIGTERM)
		if err != nil {
			return err
		}

		// Try to gracefully shutdown the process by sending TERMINATE
		// first. After 5 seconds KILL will be sent.
		stepDuration := gracePeriod / 10
		for i := 0; i < 10; i++ {
			time.Sleep(stepDuration)
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				return nil // ### return, success ###
			}
		}
	}

	return proc.Kill()
}
