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

package tgo

import (
	"log"
	"os"
	"runtime"
	"time"
)

// ShutdownCallback holds the function that is called when RecoverShutdown detects
// a panic and could recover. By default this functions sends an os.Interrupt
// signal to the process. This function can be overwritten for customization.
var ShutdownCallback = func() {
	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(os.Interrupt)
}

// ReturnAfter calls a function. If that function does not return after the
// given limit, the function returns regardless of the callback being done or
// not. This guarantees the call to finish before or at the given limit.
func ReturnAfter(runtimeLimit time.Duration, callback func()) bool {
	timeout := time.NewTimer(runtimeLimit)
	callOk := make(chan bool)

	go func() {
		callback()
		timeout.Stop()
		callOk <- true
	}()

	select {
	case <-timeout.C:
		return false

	case <-callOk:
		return true
	}
}

// RecoverShutdown will trigger a shutdown via os.Interrupt if a panic was issued.
// A callstack will be printed like with RecoverTrace().
// Typically used as "defer RecoverShutdown()".
func RecoverShutdown() {
	if r := recover(); r != nil {
		log.Printf("Panic shutdown: %s", r)
		for i := 0; i < 10; i++ {
			_, file, line, ok := runtime.Caller(i)
			if !ok {
				break // ### break, could not retrieve ###
			}
			log.Printf("\t%s:%d", file, line)
		}
		ShutdownCallback()
	}
}

// RecoverTrace will trigger a stack trace when a panic was recovered by this
// function. Typically used as "defer RecoverTrace()".
func RecoverTrace() {
	if r := recover(); r != nil {
		log.Printf("Panic ignored: %s", r)
		for i := 0; i < 10; i++ {
			_, file, line, ok := runtime.Caller(i)
			if !ok {
				break // ### break, could not retrieve ###
			}
			log.Printf("\t%s:%d", file, line)
		}
	}
}

// WithRecover can be used instead of RecoverTrace when using a function
// without any parameters. E.g. "go WithRecover(myFunction)"
func WithRecover(callback func()) {
	defer RecoverTrace()
	callback()
}

// WithRecoverShutdown can be used instead of RecoverShutdown when using a function
// without any parameters. E.g. "go WithRecoverShutdown(myFunction)"
func WithRecoverShutdown(callback func()) {
	defer RecoverShutdown()
	callback()
}
