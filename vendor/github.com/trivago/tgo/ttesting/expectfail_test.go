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

func TestBoolAndNilFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	expect.False(silent.True(false))
	expect.False(silent.False(true))

	var emptyFunc func()

	expect.False(silent.NotNil(nil))
	expect.False(silent.NotNil([]int(nil)))
	expect.False(silent.NotNil(map[string]string(nil)))
	expect.False(silent.NotNil(interface{}(nil)))
	expect.False(silent.NotNil(chan int(nil)))
	expect.False(silent.NotNil(emptyFunc))

	expect.False(silent.Nil(1))
	expect.False(silent.Nil(struct{}{}))
	expect.False(silent.Nil([]int{1}))
	expect.False(silent.Nil(map[string]string{"a": "a"}))
	expect.False(silent.Nil(make(chan int)))
	expect.False(silent.Nil(TestBoolAndNil))
}

func TestContainsFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	expect.False(silent.Contains([]int{1, 2, 3}, 4))

	expect.False(silent.Contains("foobar", "xxx"))

	expect.False(silent.ContainsN([]int{1, 1, 3}, 1, 1))
	expect.False(silent.ContainsN([]int{1, 2, 2}, 2, 3))
	expect.False(silent.ContainsN([]int{3, 2, 3}, 3, 0))

	expect.False(silent.ContainsN("foobarfoo", "foo", 1))
	expect.False(silent.ContainsN("foobarbar", "bar", 3))
}

func TestMapFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	values := map[string]int{
		"a": 1,
		"b": 10,
	}

	expect.False(silent.MapSet(values, "foo"))
	expect.False(silent.MapNotSet(values, "a"))

	expect.False(silent.MapEqual(values, "a", 0))
	expect.False(silent.MapNeq(values, "a", 1))

	expect.False(silent.MapLess(values, "a", 1))
	expect.False(silent.MapLess(values, "b", 10))
	expect.False(silent.MapLeq(values, "a", 2))
	expect.False(silent.MapLeq(values, "b", 11))

	expect.False(silent.MapLess(values, "a", 2))
	expect.False(silent.MapLess(values, "b", 11))

	expect.False(silent.MapGreater(values, "a", 1))
	expect.False(silent.MapGreater(values, "b", 10))
	expect.False(silent.MapGeq(values, "a", 0))
	expect.False(silent.MapGeq(values, "b", 9))

	expect.False(silent.MapGreater(values, "a", 0))
	expect.False(silent.MapGreater(values, "b", 9))
}

func TestEqualFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	expect.False(silent.Equal(int8(1), int8(10)))
	expect.False(silent.Equal(uint8(1), uint8(10)))
	expect.False(silent.Equal(int16(1), int16(10)))
	expect.False(silent.Equal(uint16(1), uint16(10)))
	expect.False(silent.Equal(int32(1), int32(10)))
	expect.False(silent.Equal(uint32(1), uint32(10)))
	expect.False(silent.Equal(int64(1), int64(10)))
	expect.False(silent.Equal(uint64(1), uint64(10)))
	expect.False(silent.Equal(byte(1), byte(10)))
	expect.False(silent.Equal(int(1), int(10)))
	expect.False(silent.Equal(float32(1), float32(10)))
	expect.False(silent.Equal(float64(1), float64(10)))
	expect.False(silent.Equal("foo", "bar"))
	expect.False(silent.Equal([]int{1, 2, 3}, []int{3, 2, 1}))
	expect.False(silent.Equal(map[string]int{"a": 1, "b": 2, "c": 3}, map[string]int{"a": 3, "b": 2, "c": 1}))
}

func TestNeqFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	expect.False(silent.Neq(int8(10), int8(10)))
	expect.False(silent.Neq(uint8(10), uint8(10)))
	expect.False(silent.Neq(int16(10), int16(10)))
	expect.False(silent.Neq(uint16(10), uint16(10)))
	expect.False(silent.Neq(int32(10), int32(10)))
	expect.False(silent.Neq(uint32(10), uint32(10)))
	expect.False(silent.Neq(int64(10), int64(10)))
	expect.False(silent.Neq(uint64(10), uint64(10)))
	expect.False(silent.Neq(byte(10), byte(10)))
	expect.False(silent.Neq(int(10), int(10)))
	expect.False(silent.Neq(float32(10), float32(10)))
	expect.False(silent.Neq(float64(10), float64(10)))
	expect.False(silent.Neq("foo", "foo"))
	expect.False(silent.Neq([]int{1, 2, 3}, []int{1, 2, 3}))
	expect.False(silent.Neq(map[string]int{"a": 1, "b": 2, "c": 3}, map[string]int{"a": 1, "b": 2, "c": 3}))
}

func TestLessFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	expect.False(silent.Geq(int8(1), int8(10)))
	expect.False(silent.Geq(uint8(1), uint8(10)))
	expect.False(silent.Geq(int16(1), int16(10)))
	expect.False(silent.Geq(uint16(1), uint16(10)))
	expect.False(silent.Geq(int32(1), int32(10)))
	expect.False(silent.Geq(uint32(1), uint32(10)))
	expect.False(silent.Geq(int64(1), int64(10)))
	expect.False(silent.Geq(uint64(1), uint64(10)))
	expect.False(silent.Geq(byte(1), byte(10)))
	expect.False(silent.Geq(int(1), int(10)))
	expect.False(silent.Geq(float32(1), float32(10)))
	expect.False(silent.Geq(float64(1), float64(10)))
}

func TestLeqFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	expect.False(silent.Greater(int8(1), int8(10)))
	expect.False(silent.Greater(uint8(1), uint8(10)))
	expect.False(silent.Greater(int16(1), int16(10)))
	expect.False(silent.Greater(uint16(1), uint16(10)))
	expect.False(silent.Greater(int32(1), int32(10)))
	expect.False(silent.Greater(uint32(1), uint32(10)))
	expect.False(silent.Greater(int64(1), int64(10)))
	expect.False(silent.Greater(uint64(1), uint64(10)))
	expect.False(silent.Greater(byte(1), byte(10)))
	expect.False(silent.Greater(int(1), int(10)))
	expect.False(silent.Greater(float32(1), float32(10)))
	expect.False(silent.Greater(float64(1), float64(10)))

	expect.False(silent.Greater(int8(10), int8(10)))
	expect.False(silent.Greater(uint8(10), uint8(10)))
	expect.False(silent.Greater(int16(10), int16(10)))
	expect.False(silent.Greater(uint16(10), uint16(10)))
	expect.False(silent.Greater(int32(10), int32(10)))
	expect.False(silent.Greater(uint32(10), uint32(10)))
	expect.False(silent.Greater(int64(10), int64(10)))
	expect.False(silent.Greater(uint64(10), uint64(10)))
	expect.False(silent.Greater(byte(10), byte(10)))
	expect.False(silent.Greater(int(10), int(10)))
	expect.False(silent.Greater(float32(10), float32(10)))
	expect.False(silent.Greater(float64(10), float64(10)))
}

func TestGreaterFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	expect.False(silent.Leq(int8(10), int8(1)))
	expect.False(silent.Leq(uint8(10), uint8(1)))
	expect.False(silent.Leq(int16(10), int16(1)))
	expect.False(silent.Leq(uint16(10), uint16(1)))
	expect.False(silent.Leq(int32(10), int32(1)))
	expect.False(silent.Leq(uint32(10), uint32(1)))
	expect.False(silent.Leq(int64(10), int64(1)))
	expect.False(silent.Leq(uint64(10), uint64(1)))
	expect.False(silent.Leq(byte(10), byte(1)))
	expect.False(silent.Leq(int(10), int(1)))
	expect.False(silent.Leq(float32(10), float32(1)))
	expect.False(silent.Leq(float64(10), float64(1)))
}

func TestGeqFail(t *testing.T) {
	expect := NewExpect(t)
	silent := NewSilentExpect(t)

	expect.False(silent.Less(int8(10), int8(1)))
	expect.False(silent.Less(uint8(10), uint8(1)))
	expect.False(silent.Less(int16(10), int16(1)))
	expect.False(silent.Less(uint16(10), uint16(1)))
	expect.False(silent.Less(int32(10), int32(1)))
	expect.False(silent.Less(uint32(10), uint32(1)))
	expect.False(silent.Less(int64(10), int64(1)))
	expect.False(silent.Less(uint64(10), uint64(1)))
	expect.False(silent.Less(byte(10), byte(1)))
	expect.False(silent.Less(int(10), int(1)))
	expect.False(silent.Less(float32(10), float32(1)))
	expect.False(silent.Less(float64(10), float64(1)))

	expect.False(silent.Less(int8(10), int8(10)))
	expect.False(silent.Less(uint8(10), uint8(10)))
	expect.False(silent.Less(int16(10), int16(10)))
	expect.False(silent.Less(uint16(10), uint16(10)))
	expect.False(silent.Less(int32(10), int32(10)))
	expect.False(silent.Less(uint32(10), uint32(10)))
	expect.False(silent.Less(int64(10), int64(10)))
	expect.False(silent.Less(uint64(10), uint64(10)))
	expect.False(silent.Less(byte(10), byte(10)))
	expect.False(silent.Less(int(10), int(10)))
	expect.False(silent.Less(float32(10), float32(10)))
	expect.False(silent.Less(float64(10), float64(10)))
}
