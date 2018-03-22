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
	"sync/atomic"
	"time"
)

// WaitGroup is a replacement for sync/waitgroup that allows the internal counter
// to go negative. This version allows a missed Done to be recovered but will
// make it a lot harder to detect missing Done or Add calls.
// Use only where needed.
type WaitGroup struct {
	counter int32
}

// Active returns true if the counter is > 0
func (wg *WaitGroup) Active() bool {
	return atomic.LoadInt32(&wg.counter) > 0
}

// Inc is the shorthand version for Add(1)
func (wg *WaitGroup) Inc() {
	atomic.AddInt32(&wg.counter, 1)
}

// Add increments the waitgroup counter by the given value.
// Delta may be negative.
func (wg *WaitGroup) Add(delta int) {
	atomic.AddInt32(&wg.counter, int32(delta))
}

// Done is the shorthand version for Add(-1)
func (wg *WaitGroup) Done() {
	atomic.AddInt32(&wg.counter, -1)
}

// Reset sets the counter to 0
func (wg *WaitGroup) Reset() {
	atomic.StoreInt32(&wg.counter, 0)
}

// IncWhenDone wait until the counter is exactly 0 and triggeres an increment
// if this is found to be true
func (wg *WaitGroup) IncWhenDone() {
	spin := NewSpinner(SpinPriorityHigh)
	for !atomic.CompareAndSwapInt32(&wg.counter, 0, 1) {
		spin.Yield()
	}
}

// Wait blocks until the counter is 0 or less.
func (wg *WaitGroup) Wait() {
	spin := NewSpinner(SpinPriorityHigh)
	for wg.Active() {
		spin.Yield()
	}
}

// WaitFor blocks until the counter is 0 or less. If the block takes longer than
// the given timeout, WaitFor will return false. If duration is 0, Wait is called.
func (wg *WaitGroup) WaitFor(timeout time.Duration) bool {
	if timeout == time.Duration(0) {
		wg.Wait()
		return true // ### return, always true ###
	}

	start := time.Now()
	spin := NewSpinner(SpinPriorityHigh)
	for wg.Active() {
		if time.Since(start) > timeout {
			return false // ### return, timed out ###
		}
		spin.Yield()
	}
	return true
}
