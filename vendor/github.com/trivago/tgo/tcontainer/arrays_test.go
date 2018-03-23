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
	"github.com/trivago/tgo/ttesting"
	"math/rand"
	"testing"
)

func TestInt64SliceSort(t *testing.T) {
	expect := ttesting.NewExpect(t)
	array := make(Int64Slice, 10)

	for i := range array {
		array[i] = rand.Int63()
	}

	// force not sorted
	array[0] = 1
	array[9] = 0

	expect.False(array.IsSorted())
	array.Sort()
	expect.True(array.IsSorted())
}

func TestInt64SliceSet(t *testing.T) {
	expect := ttesting.NewExpect(t)
	array := make(Int64Slice, 10)

	array.Set(1)

	for _, v := range array {
		expect.Equal(int64(1), v)
	}
}

func TestUint64SliceSort(t *testing.T) {
	expect := ttesting.NewExpect(t)
	array := make(Uint64Slice, 10)

	for i := range array {
		array[i] = uint64(rand.Int63())
	}

	// force not sorted
	array[0] = 1
	array[9] = 0

	expect.False(array.IsSorted())
	array.Sort()
	expect.True(array.IsSorted())
}

func TestUint64SliceSet(t *testing.T) {
	expect := ttesting.NewExpect(t)
	array := make(Uint64Slice, 10)

	array.Set(1)

	for _, v := range array {
		expect.Equal(uint64(1), v)
	}
}

func TestFloat32SliceSort(t *testing.T) {
	expect := ttesting.NewExpect(t)
	array := make(Float32Slice, 10)

	for i := range array {
		array[i] = rand.Float32()
	}

	// force not sorted
	array[0] = 1
	array[9] = 0

	expect.False(array.IsSorted())
	array.Sort()
	expect.True(array.IsSorted())
}

func TestFloat32SliceSet(t *testing.T) {
	expect := ttesting.NewExpect(t)
	array := make(Float32Slice, 10)

	array.Set(float32(1))

	for _, v := range array {
		expect.Equal(float32(1), v)
	}
}
