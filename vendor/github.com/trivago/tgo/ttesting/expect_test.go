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

package ttesting

import (
	"testing"
)

func TestBoolAndNil(t *testing.T) {
	expect := NewExpect(t)

	expect.True(true)
	expect.False(false)

	var emptyFunc func()

	expect.Nil(nil)
	expect.Nil([]int(nil))
	expect.Nil(map[string]string(nil))
	expect.Nil(interface{}(nil))
	expect.Nil(chan int(nil))
	expect.Nil(emptyFunc)

	expect.NotNil(1)
	expect.NotNil(struct{}{})
	expect.NotNil([]int{1})
	expect.NotNil(map[string]string{"a": "a"})
	expect.NotNil(make(chan int))
	expect.NotNil(TestBoolAndNil)
}

func TestContains(t *testing.T) {
	expect := NewExpect(t)

	expect.Contains([]int{1, 2, 3}, 1)
	expect.Contains([]int{1, 2, 3}, 2)
	expect.Contains([]int{1, 2, 3}, 3)

	expect.Contains("foobar", "foo")
	expect.Contains("foobar", "bar")

	expect.ContainsN([]int{1, 1, 3}, 1, 2)
	expect.ContainsN([]int{1, 2, 2}, 2, 2)
	expect.ContainsN([]int{3, 2, 3}, 3, 2)

	expect.ContainsN("foobarfoo", "foo", 2)
	expect.ContainsN("foobarbar", "bar", 2)
}

func TestMap(t *testing.T) {
	expect := NewExpect(t)

	values := map[string]int{
		"a": 1,
		"b": 10,
	}

	expect.MapSet(values, "a")
	expect.MapNotSet(values, "foo")

	expect.MapEqual(values, "a", 1)
	expect.MapNeq(values, "a", 0)

	expect.MapGeq(values, "a", 1)
	expect.MapGeq(values, "b", 10)
	expect.MapGeq(values, "a", 2)
	expect.MapGeq(values, "b", 11)

	expect.MapGreater(values, "a", 2)
	expect.MapGreater(values, "b", 11)

	expect.MapLeq(values, "a", 1)
	expect.MapLeq(values, "b", 10)
	expect.MapLeq(values, "a", 0)
	expect.MapLeq(values, "b", 9)

	expect.MapLess(values, "a", 0)
	expect.MapLess(values, "b", 9)
}

func TestEqual(t *testing.T) {
	expect := NewExpect(t)

	expect.Equal(int8(10), int8(10))
	expect.Equal(uint8(10), uint8(10))
	expect.Equal(int16(10), int16(10))
	expect.Equal(uint16(10), uint16(10))
	expect.Equal(int32(10), int32(10))
	expect.Equal(uint32(10), uint32(10))
	expect.Equal(int64(10), int64(10))
	expect.Equal(uint64(10), uint64(10))
	expect.Equal(byte(10), byte(10))
	expect.Equal(int(10), int(10))
	expect.Equal(float32(10), float32(10))
	expect.Equal(float64(10), float64(10))
	expect.Equal("foo", "foo")
	expect.Equal([]int{1, 2, 3}, []int{1, 2, 3})
	expect.Equal(map[string]int{"a": 1, "b": 2, "c": 3}, map[string]int{"a": 1, "b": 2, "c": 3})
}

func TestNeq(t *testing.T) {
	expect := NewExpect(t)

	expect.Neq(int8(1), int8(10))
	expect.Neq(uint8(1), uint8(10))
	expect.Neq(int16(1), int16(10))
	expect.Neq(uint16(1), uint16(10))
	expect.Neq(int32(1), int32(10))
	expect.Neq(uint32(1), uint32(10))
	expect.Neq(int64(1), int64(10))
	expect.Neq(uint64(1), uint64(10))
	expect.Neq(byte(1), byte(10))
	expect.Neq(int(1), int(10))
	expect.Neq(float32(1), float32(10))
	expect.Neq(float64(1), float64(10))
	expect.Neq("foo", "bar")
	expect.Neq([]int{1, 2, 3}, []int{3, 2, 1})
	expect.Neq(map[string]int{"a": 1, "b": 2, "c": 3}, map[string]int{"a": 3, "b": 2, "c": 1})
}

func TestLess(t *testing.T) {
	expect := NewExpect(t)

	expect.Less(int8(1), int8(10))
	expect.Less(uint8(1), uint8(10))
	expect.Less(int16(1), int16(10))
	expect.Less(uint16(1), uint16(10))
	expect.Less(int32(1), int32(10))
	expect.Less(uint32(1), uint32(10))
	expect.Less(int64(1), int64(10))
	expect.Less(uint64(1), uint64(10))
	expect.Less(byte(1), byte(10))
	expect.Less(int(1), int(10))
	expect.Less(float32(1), float32(10))
	expect.Less(float64(1), float64(10))
}

func TestLeq(t *testing.T) {
	expect := NewExpect(t)

	expect.Leq(int8(1), int8(10))
	expect.Leq(uint8(1), uint8(10))
	expect.Leq(int16(1), int16(10))
	expect.Leq(uint16(1), uint16(10))
	expect.Leq(int32(1), int32(10))
	expect.Leq(uint32(1), uint32(10))
	expect.Leq(int64(1), int64(10))
	expect.Leq(uint64(1), uint64(10))
	expect.Leq(byte(1), byte(10))
	expect.Leq(int(1), int(10))
	expect.Leq(float32(1), float32(10))
	expect.Leq(float64(1), float64(10))

	expect.Leq(int8(10), int8(10))
	expect.Leq(uint8(10), uint8(10))
	expect.Leq(int16(10), int16(10))
	expect.Leq(uint16(10), uint16(10))
	expect.Leq(int32(10), int32(10))
	expect.Leq(uint32(10), uint32(10))
	expect.Leq(int64(10), int64(10))
	expect.Leq(uint64(10), uint64(10))
	expect.Leq(byte(10), byte(10))
	expect.Leq(int(10), int(10))
	expect.Leq(float32(10), float32(10))
	expect.Leq(float64(10), float64(10))
}

func TestGreater(t *testing.T) {
	expect := NewExpect(t)

	expect.Greater(int8(10), int8(1))
	expect.Greater(uint8(10), uint8(1))
	expect.Greater(int16(10), int16(1))
	expect.Greater(uint16(10), uint16(1))
	expect.Greater(int32(10), int32(1))
	expect.Greater(uint32(10), uint32(1))
	expect.Greater(int64(10), int64(1))
	expect.Greater(uint64(10), uint64(1))
	expect.Greater(byte(10), byte(1))
	expect.Greater(int(10), int(1))
	expect.Greater(float32(10), float32(1))
	expect.Greater(float64(10), float64(1))
}

func TestGeq(t *testing.T) {
	expect := NewExpect(t)

	expect.Geq(int8(10), int8(1))
	expect.Geq(uint8(10), uint8(1))
	expect.Geq(int16(10), int16(1))
	expect.Geq(uint16(10), uint16(1))
	expect.Geq(int32(10), int32(1))
	expect.Geq(uint32(10), uint32(1))
	expect.Geq(int64(10), int64(1))
	expect.Geq(uint64(10), uint64(1))
	expect.Geq(byte(10), byte(1))
	expect.Geq(int(10), int(1))
	expect.Geq(float32(10), float32(1))
	expect.Geq(float64(10), float64(1))

	expect.Geq(int8(10), int8(10))
	expect.Geq(uint8(10), uint8(10))
	expect.Geq(int16(10), int16(10))
	expect.Geq(uint16(10), uint16(10))
	expect.Geq(int32(10), int32(10))
	expect.Geq(uint32(10), uint32(10))
	expect.Geq(int64(10), int64(10))
	expect.Geq(uint64(10), uint64(10))
	expect.Geq(byte(10), byte(10))
	expect.Geq(int(10), int(10))
	expect.Geq(float32(10), float32(10))
	expect.Geq(float64(10), float64(10))
}
