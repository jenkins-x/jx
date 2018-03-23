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
	"github.com/trivago/tgo/tmath"
	"sync/atomic"
)

// Stack implements a simple, growing, lockfree stack.
// The main idea is to use the sign bit of the head index as a mutex.
// If the index is negative, the stack is locked so we need to spin.
// If the index is non-negative the stack is unlocked and we can write or read.
type Stack struct {
	data   []interface{}
	growBy int
	head   *uint32
	spin   Spinner
}

const unlockMask = uint32(0x7FFFFFFF)
const lockMask = uint32(0x80000000)

// NewStack creates a new stack with the given initial size.
// The given size will also be used as grow size.
// SpinPriorityMedium is used to initialize the spinner.
func NewStack(size int) Stack {
	return NewStackWithSpinnerAndGrowSize(size, size, NewSpinner(SpinPriorityMedium))
}

// NewStackWithGrowSize allows to pass a custom grow size to the stack.
// SpinPriorityMedium is used to initialize the spinner.
func NewStackWithGrowSize(size, grow int) Stack {
	return NewStackWithSpinnerAndGrowSize(size, grow, NewSpinner(SpinPriorityMedium))
}

// NewStackWithSpinner allows to pass a custom spinner to the stack.
// The given size will also be used as grow size.
func NewStackWithSpinner(size int, spin Spinner) Stack {
	return NewStackWithSpinnerAndGrowSize(size, size, spin)
}

// NewStackWithSpinnerAndGrowSize allows to fully configure the new stack.
func NewStackWithSpinnerAndGrowSize(size, grow int, spin Spinner) Stack {
	return Stack{
		data:   make([]interface{}, tmath.MinI(size, 1)),
		growBy: tmath.MinI(grow, 1),
		head:   new(uint32),
		spin:   spin,
	}
}

// Len returns the number of elements on the stack.
// Please note that this value can be highly unreliable in multithreaded
// environments as this is only a snapshot of the state at calltime.
func (s *Stack) Len() int {
	return int(atomic.LoadUint32(s.head) & unlockMask)
}

// Pop retrieves the topmost element from the stack.
// A LimitError is returned when the stack is empty.
func (s *Stack) Pop() (interface{}, error) {
	spin := s.spin
	for {
		head := atomic.LoadUint32(s.head)
		unlockedHead := head & unlockMask
		lockedHead := head | lockMask

		// Always work with unlocked head as head might be locked
		if unlockedHead == 0 {
			return nil, LimitError{"Stack is empty"}
		}

		if atomic.CompareAndSwapUint32(s.head, unlockedHead, lockedHead) {
			data := s.data[unlockedHead-1]             // copy data
			atomic.StoreUint32(s.head, unlockedHead-1) // unlock
			return data, nil                           // ### return ###
		}

		spin.Yield()
	}
}

// Push adds an element to the top of the stack.
// When the stack's capacity is reached the storage grows as defined during
// construction. If the stack reaches 2^31 elements it is considered full
// and will return an LimitError.
func (s *Stack) Push(v interface{}) error {
	spin := s.spin
	for {
		head := atomic.LoadUint32(s.head)
		unlockedHead := head & unlockMask
		lockedHead := head | lockMask

		// Always work with unlocked head as head might be locked
		if unlockedHead == unlockMask {
			return LimitError{"Stack is full"}
		}

		if atomic.CompareAndSwapUint32(s.head, unlockedHead, lockedHead) {
			if unlockedHead == uint32(len(s.data)) {
				// Grow stack
				old := s.data
				s.data = make([]interface{}, len(s.data)+s.growBy)
				copy(s.data, old)
			}

			s.data[unlockedHead] = v                   // write to new head
			atomic.StoreUint32(s.head, unlockedHead+1) // unlock
			return nil                                 // ### return ###
		}

		spin.Yield()
	}
}
