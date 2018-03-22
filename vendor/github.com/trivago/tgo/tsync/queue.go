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

// Queue implements a multi-producer, multi-consumer, fair, lockfree queue.
// Please note that while this queue is faster than a chan interface{} it
// is not faster than a channel of a specific type due to the overhead
// that interface{} brings.
//
// How it works:
// Given a queue of capacity 7, we have two write indexes "p" and "n" for
// process and next.
//
//    reader n/p
//    v
// [ |1|2|3| | | | ]
//    ^     ^
//    p     n
//
// Here threads 1,2 and 3 are writing to slots 1,2,3. When 3 is done
// writing it has to wait for p to move to slot 3. Same for slot 2.
// When 1 is done writing it immediatly moves p to slot 2, making it
// possible for 2 to continue. Same for slot 3.
// While p is still on 1 the "reader n" can be fetched but Pop will
// wait until p has moved on. If "reader n" is done, "reader p" is
// moved in similar fashion as p for writes.
// This implements a FIFO queue for writes and reads and makes sure
// that no incomplete reads or overwrites occur.
type Queue struct {
	write    queueAccess
	read     queueAccess
	capacity uint64
	locked   *int32
	spin     Spinner
	items    []interface{}
}

// NewQueue creates a new queue with medium spinning priority
func NewQueue(capacity uint32) Queue {
	return NewQueueWithSpinner(capacity, NewSpinner(SpinPriorityMedium))
}

// NewQueueWithSpinner allows to set the spinning priority of the queue to
// be created.
func NewQueueWithSpinner(capacity uint32, spinner Spinner) Queue {
	return Queue{
		items:    make([]interface{}, capacity),
		read:     newQueueAccess(),
		write:    newQueueAccess(),
		locked:   new(int32),
		capacity: uint64(capacity),
		spin:     spinner,
	}
}

// Push adds an item to the queue. This call may block if the queue is full.
// An error is returned when the queue is locked.
func (q *Queue) Push(item interface{}) error {
	if atomic.LoadInt32(q.locked) == 1 {
		return LockedError{"Queue is locked"} // ### return, closed ###
	}

	// Get ticket and slot
	ticket := atomic.AddUint64(q.write.next, 1) - 1
	slot := ticket % q.capacity
	spin := q.spin

	// Wait for pending reads on slot
	for ticket-atomic.LoadUint64(q.read.processing) >= q.capacity {
		spin.Yield()
	}

	q.items[slot] = item

	// Wait for previous writers to finish writing
	for ticket != atomic.LoadUint64(q.write.processing) {
		spin.Yield()
	}
	atomic.AddUint64(q.write.processing, 1)
	return nil
}

// Pop removes an item from the queue. This call may block if the queue is
// empty. If the queue is drained Pop() will not block and return nil.
func (q *Queue) Pop() interface{} {
	// Drained?
	if atomic.LoadInt32(q.locked) == 1 &&
		atomic.LoadUint64(q.write.processing) == atomic.LoadUint64(q.read.processing) {
		return nil // ### return, closed and no items ###
	}

	// Get ticket andd slot
	ticket := atomic.AddUint64(q.read.next, 1) - 1
	slot := ticket % q.capacity
	spin := q.spin

	// Wait for slot to be written to
	for ticket >= atomic.LoadUint64(q.write.processing) {
		spin.Yield()
		// Drained?
		if atomic.LoadInt32(q.locked) == 1 &&
			atomic.LoadUint64(q.write.processing) == atomic.LoadUint64(q.read.processing) {
			return nil // ### return, closed while spinning ###
		}
	}

	item := q.items[slot]

	// Wait for other reads to finish
	for ticket != atomic.LoadUint64(q.read.processing) {
		spin.Yield()
	}
	atomic.AddUint64(q.read.processing, 1)
	return item
}

// Close blocks the queue from write access. It also allows Pop() to return
// false as a second return value
func (q *Queue) Close() {
	atomic.StoreInt32(q.locked, 1)
}

// Reopen unblocks the queue to allow write access again.
func (q *Queue) Reopen() {
	atomic.StoreInt32(q.locked, 0)
}

// IsClosed returns true if Close() has been called.
func (q *Queue) IsClosed() bool {
	return atomic.LoadInt32(q.locked) == 1
}

// IsEmpty returns true if there is no item in the queue to be processed.
// Please note that this state is extremely volatile unless IsClosed
// returned true.
func (q *Queue) IsEmpty() bool {
	return atomic.LoadUint64(q.write.processing) == atomic.LoadUint64(q.read.processing)
}

// IsDrained combines IsClosed and IsEmpty.
func (q *Queue) IsDrained() bool {
	return q.IsClosed() && q.IsEmpty()
}

// Queue access encapsulates the two-index-access pattern for this queue.
// If one or both indices overflow there will be errors. This happens after
// 18 * 10^18 writes aka never if you are not doing more than 10^11 writes
// per second (overflow after ~694 days).
// Just to put this into perspective - an Intel Core i7 5960X at 3.5 Ghz can
// do 336 * 10^9 ops per second (336ops/ns). A call to push allone costs about
// 600-700µs so you won't come anywhere close to this.
type queueAccess struct {
	processing *uint64
	next       *uint64
}

func newQueueAccess() queueAccess {
	return queueAccess{
		processing: new(uint64),
		next:       new(uint64),
	}
}
