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
	"github.com/trivago/tgo/ttesting"
	"os"
	"testing"
)

func TestPidfile(t *testing.T) {
	expect := ttesting.NewExpect(t)

	pidFile := "/tmp/__tgo_tos_test.pid"
	pid := os.Getpid()

	err := WritePidFileForced(pid, pidFile)
	expect.NoError(err)

	_, err = GetProcFromFile(pidFile)
	expect.NoError(err)
	os.Remove(pidFile)

	err = WritePidFile(pid, pidFile)
	expect.NoError(err)

	_, err = GetProcFromFile(pidFile)
	expect.NoError(err)
	os.Remove(pidFile)
}

/*func TestTerminate(t *testing.T) {
	expect := ttesting.NewExpect(t)

	proc, err := os.FindProcess(os.Getpid())
	expect.NoError(err)

	signalQueue := make(chan os.Signal, 1)
	signal.Notify(signalQueue, syscall.SIGTERM, syscall.SIGKILL)

	termCalled := false
	killCalled := false

	go func() {
		for {
			sig, more := <-signalQueue
			switch {
			case sig == syscall.SIGTERM:
				termCalled = true
			case sig == syscall.SIGKILL:
				killCalled = true
			case !more:
				return
			}
		}
	}()

	// TODO: Capture kill without terminating the test...
	err = Terminate(proc, time.Second)
	time.Sleep(time.Second)

	signal.Reset(syscall.SIGTERM, syscall.SIGKILL)
	close(signalQueue)

	expect.True(termCalled)
	expect.True(killCalled)
	expect.Nil(err)
}*/
