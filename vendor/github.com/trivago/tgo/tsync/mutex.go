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
)

// Mutex is a lightweight, spinner based mutex implementation, extending the
// standard go mutex by the possibility to query the mutex' state and by adding
// a TryLock function.
type Mutex struct {
	state    *int32
	priority SpinPriority
}

const (
	mutexUnlocked = int32(0)
	mutexLocked   = int32(1)
)

// NewMutex creates a new mutex with the given spin priority used during Lock.
func NewMutex(priority SpinPriority) *Mutex {
	return &Mutex{
		state:    new(int32),
		priority: priority,
	}
}

// Lock blocks (spins) until the lock becomes available
func (m *Mutex) Lock() {
	spin := NewSpinner(m.priority)
	for !m.TryLock() {
		spin.Yield()
	}
}

// TryLock tries to acquire a lock and returns true if it succeeds. This
// function does not block.
func (m *Mutex) TryLock() bool {
	return atomic.CompareAndSwapInt32(m.state, mutexUnlocked, mutexLocked)
}

// Unlock unblocks one routine waiting on lock.
func (m *Mutex) Unlock() {
	atomic.StoreInt32(m.state, mutexUnlocked)
}

// IsLocked returns the state of this mutex. The result of this function might
// change directly after call so it should only be used in situations where
// this fact is not considered problematic.
func (m *Mutex) IsLocked() bool {
	return atomic.LoadInt32(m.state) != mutexUnlocked
}
