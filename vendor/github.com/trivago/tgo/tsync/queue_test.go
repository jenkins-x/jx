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
	"github.com/trivago/tgo/ttesting"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueueFunctionality(t *testing.T) {
	expect := ttesting.NewExpect(t)
	q := NewQueue(1)

	expect.NonBlocking(time.Second, func() { expect.NoError(q.Push(1)) })
	expect.False(q.IsEmpty())

	v := q.Pop()
	expect.Equal(1, v)
	expect.True(q.IsEmpty())
	expect.False(q.IsDrained())

	expect.NonBlocking(time.Second, func() { expect.NoError(q.Push(2)) })
	q.Close()

	v = q.Pop()
	expect.Equal(2, v)
	expect.True(q.IsEmpty())
	expect.True(q.IsDrained())

	err := q.Push(3)
	expect.OfType(LockedError{}, err)
}

func TestQueueConcurrency(t *testing.T) {
	expect := ttesting.NewExpect(t)
	q := NewQueueWithSpinner(100, NewCustomSpinner(time.Millisecond))

	writer := WaitGroup{}
	reader := WaitGroup{}
	numSamples := 100

	results := make([]*uint64, 20)
	writes := new(uint32)

	for i := 0; i < len(results); i++ {
		results[i] = new(uint64)
		idx := i
		// Start writer
		writer.Add(1)
		go func() {
			defer writer.Done()
			for m := 0; m < numSamples; m++ {
				expect.NoError(q.Push(idx))
				atomic.AddUint32(writes, 1)
				runtime.Gosched()
			}
		}()
		// start reader
		reader.Add(1)
		go func() {
			defer reader.Done()
			for !q.IsDrained() {
				value := q.Pop()
				if value != nil {
					idx, _ := value.(int)
					atomic.AddUint64(results[idx], 1)
					runtime.Gosched()
				}
			}
		}()
	}

	// Give them some time
	writer.WaitFor(time.Second * 5)
	expect.Equal(int32(0), writer.counter)
	expect.Equal(len(results)*numSamples, int(atomic.LoadUint32(writes)))

	q.Close()
	reader.WaitFor(time.Second * 5)
	expect.Equal(int32(0), reader.counter)

	// Check results
	for i := 0; i < len(results); i++ {
		expect.Equal(uint64(numSamples), atomic.LoadUint64(results[i]))
	}
}

func BenchmarkQueuePush(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := NewQueue(100000)
		for c := 0; c < 100000; c++ {
			q.Push(123)
		}
	}
}

func BenchmarkChannelPush(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := make(chan interface{}, 100000)
		for c := 0; c < 100000; c++ {
			q <- 123
		}
	}
}

func BenchmarkQueuePop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := NewQueue(100000)
		for c := 0; c < 100000; c++ {
			q.Push(123)
		}

		for c := 0; c < 100000; c++ {
			q.Pop()
		}
	}
}

func BenchmarkChannelPop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := make(chan interface{}, 100000)
		for c := 0; c < 100000; c++ {
			q <- 123
		}

		for c := 0; c < 100000; c++ {
			<-q
		}
	}
}
