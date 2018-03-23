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
	"sync"
	"testing"
	"time"
)

func TestFanout(t *testing.T) {
	expect := ttesting.NewExpect(t)

	in := make(chan int)
	out1 := make(chan int)
	out2 := make(chan int)
	out3 := make(chan int)

	fanoutFunc := new(sync.WaitGroup)
	fanoutFunc.Add(1)

	go func() {
		Fanout(in, out1, out2, out3)
		fanoutFunc.Done()
	}()

	in <- 1
	select {
	case d := <-out1:
		expect.Equal(1, d)
	case d := <-out2:
		expect.Equal(1, d)
	case d := <-out3:
		expect.Equal(1, d)
	case <-time.After(time.Second):
		expect.NotExecuted()
	}

	close(in)
	expect.NonBlocking(time.Second, fanoutFunc.Wait)
}

func TestFunnel(t *testing.T) {
	expect := ttesting.NewExpect(t)

	out := make(chan int)
	in1 := make(chan int, 1)
	in2 := make(chan int, 1)
	in3 := make(chan int, 1)

	funnelFunc := new(sync.WaitGroup)
	funnelFunc.Add(1)

	go func() {
		Funnel(out, in1, in2, in3)
		funnelFunc.Done()
	}()

	in1 <- 1
	in2 <- 1
	in3 <- 1

	for i := 0; i < 3; i++ {
		select {
		case d := <-out:
			expect.Equal(1, d)
		case <-time.After(time.Second):
			expect.NotExecuted()
		}
	}

	close(in1)
	close(in2)
	close(in3)
	expect.NonBlocking(time.Second, funnelFunc.Wait)
}

func TestTurnout(t *testing.T) {
	expect := ttesting.NewExpect(t)

	in1 := make(chan int, 1)
	in2 := make(chan int, 1)
	in3 := make(chan int, 1)
	out1 := make(chan int)
	out2 := make(chan int)
	out3 := make(chan int)

	turnoutFunc := new(sync.WaitGroup)
	turnoutFunc.Add(1)

	go func() {
		Turnout([]interface{}{in1, in2, in3}, []interface{}{out1, out2, out3})
		turnoutFunc.Done()
	}()

	in1 <- 1
	in2 <- 1
	in3 <- 1

	for i := 0; i < 3; i++ {
		select {
		case d := <-out1:
			expect.Equal(1, d)
		case d := <-out2:
			expect.Equal(1, d)
		case d := <-out3:
			expect.Equal(1, d)
		case <-time.After(time.Second):
			expect.NotExecuted()
		}
	}

	close(in1)
	close(in2)
	close(in3)
	expect.NonBlocking(time.Second, turnoutFunc.Wait)
}
