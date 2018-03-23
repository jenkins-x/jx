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
	"sort"
	"sync/atomic"
	"testing"
)

func TestStackFunctionality(t *testing.T) {
	expect := ttesting.NewExpect(t)
	s := NewStack(1)

	v, err := s.Pop()
	expect.Nil(v)

	s.Push(1)
	expect.Equal(1, len(s.data))

	s.Push(2)
	expect.Equal(2, len(s.data))

	v, err = s.Pop()
	expect.NoError(err)
	expect.Equal(2, v.(int))

	v, err = s.Pop()
	expect.NoError(err)
	expect.Equal(1, v.(int))

	v, err = s.Pop()
	expect.Nil(v)
}

func TestStackConcurrentPush(t *testing.T) {
	expect := ttesting.NewExpect(t)

	numRoutines := 10
	numWrites := 100

	s := NewStack(1)
	ready := WaitGroup{}
	start := WaitGroup{}
	finished := WaitGroup{}
	start.Inc()
	ready.Add(numRoutines)
	finished.Add(numRoutines)

	pool := new(int32)

	for i := 0; i < numRoutines; i++ {
		go func() {
			defer finished.Done()
			ready.Done()
			start.Wait()

			for i := 0; i < numWrites; i++ {
				num := int(atomic.AddInt32(pool, 1) - 1)
				s.Push(num)
				runtime.Gosched()
			}
		}()
	}
	ready.Wait()
	start.Done()
	finished.Wait()

	expect.Equal(numRoutines*numWrites, s.Len())

	numbers := make([]int, 0, numRoutines*numWrites)
	for _, num := range s.data {
		numbers = append(numbers, num.(int))
	}

	sort.Ints(numbers)
	expected := 0
	for _, num := range numbers {
		if !expect.Equal(expected, num) {
			expected = num
		}
		expected++
	}
}

func TestStackConcurrentPop(t *testing.T) {
	expect := ttesting.NewExpect(t)

	numRoutines := 10
	numReads := 100
	totalItems := numRoutines * numReads

	numbers := make([]int, totalItems)
	writeIdx := new(int32)

	s := NewStack(numRoutines * numReads)
	for i := 0; i < totalItems; i++ {
		s.Push(i)
	}

	ready := WaitGroup{}
	start := WaitGroup{}
	finished := WaitGroup{}
	start.Inc()
	ready.Add(numRoutines)
	finished.Add(numRoutines)

	for i := 0; i < numRoutines; i++ {
		go func() {
			defer finished.Done()
			ready.Done()
			start.Wait()

			for i := 0; i < numReads; i++ {
				idx := atomic.AddInt32(writeIdx, 1) - 1
				num, err := s.Pop()
				expect.NoError(err)
				numbers[idx] = num.(int)
				runtime.Gosched()
			}
		}()
	}
	ready.Wait()
	start.Done()
	finished.Wait()

	sort.Ints(numbers)
	expected := 0
	for _, num := range numbers {
		if !expect.Equal(expected, num) {
			expected = num
		}
		expected++
	}
}

func TestStackConcurrentPushPop(t *testing.T) {
	expect := ttesting.NewExpect(t)

	numRoutines := 10
	numWrites := 100
	numReads := 100
	totalItems := numRoutines * numWrites

	numbers := make([]int, totalItems)
	numIdx := new(int32)

	s := NewStack(1)
	ready := WaitGroup{}
	start := WaitGroup{}
	finished := WaitGroup{}
	start.Inc()
	ready.Add(numRoutines * 2)
	finished.Add(numRoutines * 2)

	pool := new(int32)

	for i := 0; i < numRoutines; i++ {
		go func() {
			defer finished.Done()
			ready.Done()
			start.Wait()

			for i := 0; i < numWrites; i++ {
				num := int(atomic.AddInt32(pool, 1) - 1)
				s.Push(num)
				runtime.Gosched()
			}
		}()

		go func() {
			defer finished.Done()
			ready.Done()
			start.Wait()

			for i := 0; i < numReads; i++ {
				num, err := s.Pop()
				if err == nil { // Some pops will fail
					idx := atomic.AddInt32(numIdx, 1) - 1
					numbers[idx] = num.(int)
				}
				runtime.Gosched()
			}
		}()
	}
	ready.Wait()
	start.Done()
	finished.Wait()

	expect.Equal(len(numbers)-int(*numIdx), s.Len())

	for int(*numIdx) < len(numbers) {
		num, err := s.Pop()
		expect.NoError(err)
		numbers[*numIdx] = num.(int)
		*numIdx++
	}

	sort.Ints(numbers)
	expected := 0
	for _, num := range numbers {
		if !expect.Equal(expected, num) {
			expected = num
		}
		expected++
	}
}
