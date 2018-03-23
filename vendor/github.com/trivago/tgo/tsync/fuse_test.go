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
	"testing"
	"time"
)

func TestFuse(t *testing.T) {
	expect := ttesting.NewExpect(t)
	fuse := NewFuse()

	expect.False(fuse.IsBurned())
	expect.NonBlocking(5*time.Millisecond, fuse.Wait)

	// Check reactivate of single wait

	start := time.Now()
	fuse.Burn()
	expect.True(fuse.IsBurned())

	time.AfterFunc(100*time.Millisecond, fuse.Activate)
	expect.NonBlocking(300*time.Millisecond, fuse.Wait)
	expect.False(fuse.IsBurned())
	expect.Less(int64(time.Since(start)), int64(150*time.Millisecond))

	// Check repeated burning

	fuse.Burn()
	expect.True(fuse.IsBurned())

	time.AfterFunc(100*time.Millisecond, fuse.Activate)
	expect.NonBlocking(300*time.Millisecond, fuse.Wait)
	expect.False(fuse.IsBurned())

	// Check reactivate of multiple waits

	fuse.Burn()
	expect.True(fuse.IsBurned())

	time.AfterFunc(100*time.Millisecond, fuse.Activate)
	go expect.NonBlocking(300*time.Millisecond, fuse.Wait)
	go expect.NonBlocking(300*time.Millisecond, fuse.Wait)
	go expect.NonBlocking(300*time.Millisecond, fuse.Wait)

	time.Sleep(400 * time.Millisecond)
	expect.False(fuse.IsBurned())
}
