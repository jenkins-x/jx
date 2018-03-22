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

package tcontainer

import (
	"bytes"
	"encoding/binary"
	"github.com/trivago/tgo/ttesting"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestBytePool(t *testing.T) {
	expect := ttesting.NewExpect(t)
	pool := NewBytePool()

	tinyMin := pool.Get(1)
	expect.Equal(tiny, cap(tinyMin))
	expect.Equal(1, len(tinyMin))

	tinyMax := pool.Get(tiny)
	expect.Equal(tiny, len(tinyMax))

	smallMin := pool.Get(tiny + 1)
	expect.Equal(small, cap(smallMin))
	expect.Equal(tiny+1, len(smallMin))

	smallMax := pool.Get(small)
	expect.Equal(small, len(smallMax))

	mediumMin := pool.Get(small + 1)
	expect.Equal(medium, cap(mediumMin))
	expect.Equal(small+1, len(mediumMin))

	mediumMax := pool.Get(medium)
	expect.Equal(medium, len(mediumMax))

	largeMin := pool.Get(medium + 1)
	expect.Equal(large, cap(largeMin))
	expect.Equal(medium+1, len(largeMin))

	largeMax := pool.Get(large)
	expect.Equal(large, len(largeMax))

	hugeMin := pool.Get(large + 1)
	expect.Equal(huge, cap(hugeMin))
	expect.Equal(large+1, len(hugeMin))

	hugeMax := pool.Get(huge)
	expect.Equal(huge, len(hugeMax))
}

func TestBytePoolParallel(t *testing.T) {
	expect := ttesting.NewExpect(t)
	pool := NewBytePool()

	start := new(sync.WaitGroup)
	done := make(chan int)

	allocate := func() {
		start.Wait()
		for {
			select {
			case <-done:
				return
			default:
				pool.Get(rand.Intn(huge))
			}
		}
	}

	start.Add(1)
	for i := 0; i < 100; i++ {
		go expect.NoPanic(allocate)
	}
	start.Done()

	time.Sleep(2 * time.Second)
	close(done)
}

func TestBytePoolUnique(t *testing.T) {
	expect := ttesting.NewExpect(t)
	pool := NewBytePool()

	numTests := tinyCount * 10
	chunks := make([][]byte, numTests)

	for i := 0; i < numTests; i++ {
		data := pool.Get(8)
		binary.PutVarint(data, int64(i))
		chunks[i] = data
	}

	for i := 0; i < numTests; i++ {
		num, err := binary.ReadVarint(bytes.NewReader(chunks[i]))
		expect.NoError(err)
		expect.Equal(int64(i), num)
	}
}
