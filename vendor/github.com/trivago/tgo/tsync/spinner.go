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

package tsync

import (
	"runtime"
	"time"
)

// Spinner is a helper struct for spinning loops.
type Spinner struct {
	count      uint32
	suspendFor time.Duration
}

// SpinPriority is used for Spinner priority enum values
type SpinPriority uint32

const (
	// SpinPrioritySuspend should be used for spinning loops that are expected
	// to wait for very long periods of time. The loop will sleep for 1 second
	// after 100 iterations.
	SpinPrioritySuspend = SpinPriority(iota)

	// SpinPriorityLow should be used for spinning loops that are expected to
	// spin for a rather long time before being able to exit.
	// After 100 loops the caller waits for 100 milliseconds.
	SpinPriorityLow = SpinPriority(iota)

	// SpinPriorityMedium should be used for spinning loops that are expected to
	// spin for a short amount of time before being able to exit.
	// After 100 loops the caller waits for 1 millisecond.
	SpinPriorityMedium = SpinPriority(iota)

	// SpinPriorityHigh should be used for spinning loops that are expected to
	// almost never spin.
	// After 100 loops the caller waits for 10 microseconds.
	SpinPriorityHigh = SpinPriority(iota)

	// SpinPriorityRealtime should be used for loops that need to run as fast
	// as possible. After 100 loops the go scheduler is triggered.
	SpinPriorityRealtime = SpinPriority(iota)
)

var (
	spinDelay = []time.Duration{
		time.Second,            // SpinPrioritySuspend
		100 * time.Millisecond, // SpinPriorityLow
		time.Millisecond,       // SpinPriorityMedium
		10 * time.Microsecond,  // SpinPriorityHigh
		time.Duration(0),       // SpinPriorityRealtime
	}
)

// NewSpinner creates a new helper for spinning loops
func NewSpinner(priority SpinPriority) Spinner {
	return Spinner{
		count:      0,
		suspendFor: spinDelay[priority],
	}
}

// NewCustomSpinner creates a new spinner with a custom delay.
func NewCustomSpinner(suspendFor time.Duration) Spinner {
	return Spinner{
		count:      0,
		suspendFor: suspendFor,
	}
}

// Yield should be called in spinning loops and will assure correct
// spin/wait/schedule behavior according to the set priority.
func (spin *Spinner) Yield() {
	if spin.count >= 100 {
		spin.count = 0
		// Always call Gosched if suspending is disabled to prevent stuck go
		// routines with GOMAXPROCS=1 and to be nice to the scheduler
		if spin.suspendFor == 0 {
			runtime.Gosched()
		} else {
			time.Sleep(spin.suspendFor)
		}
		return // ### return, suspended ###
	}

	spin.count++

}

// Reset sets the internal counter back to 0
func (spin *Spinner) Reset() {
	spin.count = 0
}
