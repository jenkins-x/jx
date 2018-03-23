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

package tstrings

import (
	"fmt"
	"github.com/trivago/tgo/ttesting"
	"testing"
)

func TestItoLen(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.Equal(1, ItoLen(0))
	expect.Equal(1, ItoLen(1))
	expect.Equal(2, ItoLen(10))
	expect.Equal(2, ItoLen(33))
	expect.Equal(3, ItoLen(100))
	expect.Equal(3, ItoLen(555))
}

func TestItob(t *testing.T) {
	expect := ttesting.NewExpect(t)
	buffer := [3]byte{'1', '1', '1'}

	Itob(0, buffer[:])
	expect.Equal("011", string(buffer[:]))

	Itob(123, buffer[:])
	expect.Equal("123", string(buffer[:]))

	Itob(45, buffer[:])
	expect.Equal("45", string(buffer[:2]))
}

func TestItobe(t *testing.T) {
	expect := ttesting.NewExpect(t)
	buffer := [3]byte{'1', '1', '1'}

	Itobe(0, buffer[:])
	expect.Equal("110", string(buffer[:]))

	Itobe(123, buffer[:])
	expect.Equal("123", string(buffer[:]))

	Itobe(45, buffer[:])
	expect.Equal("45", string(buffer[1:]))
}

func TestBtoi(t *testing.T) {
	expect := ttesting.NewExpect(t)

	result, length := Btoi([]byte("0"))
	expect.Equal(0, int(result))
	expect.Equal(1, length)

	result, length = Btoi([]byte("test"))
	expect.Equal(0, int(result))
	expect.Equal(0, length)

	result, length = Btoi([]byte("10"))
	expect.Equal(10, int(result))
	expect.Equal(2, length)

	result, length = Btoi([]byte("10x"))
	expect.Equal(10, int(result))
	expect.Equal(2, length)

	result, length = Btoi([]byte("33"))
	expect.Equal(33, int(result))
	expect.Equal(2, length)

	result, length = Btoi([]byte("100"))
	expect.Equal(100, int(result))
	expect.Equal(3, length)

	result, length = Btoi([]byte("333"))
	expect.Equal(333, int(result))
	expect.Equal(3, length)
}

func TestIndexN(t *testing.T) {
	expect := ttesting.NewExpect(t)

	testString := "a.b.c.d"
	expect.Equal(-1, IndexN(testString, ".", 4))
	expect.Equal(-1, IndexN(testString, ".", 0))
	expect.Equal(1, IndexN(testString, ".", 1))
	expect.Equal(3, IndexN(testString, ".", 2))
	expect.Equal(5, IndexN(testString, ".", 3))

	expect.Equal(-1, LastIndexN(testString, ".", 4))
	expect.Equal(-1, LastIndexN(testString, ".", 0))
	expect.Equal(5, LastIndexN(testString, ".", 1))
	expect.Equal(3, LastIndexN(testString, ".", 2))
	expect.Equal(1, LastIndexN(testString, ".", 3))
}

func TestIsIntCorrectPos(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.True(IsInt("0"))
	expect.True(IsInt("1"))
	expect.True(IsInt("2"))
	expect.True(IsInt("7"))
	expect.True(IsInt("8"))
	expect.True(IsInt("9"))

	expect.True(IsInt("000"))
	expect.True(IsInt("999"))

	expect.True(IsInt("123"))

	expect.True(IsInt("0123"))
	expect.True(IsInt("1023"))
	expect.True(IsInt("1203"))
	expect.True(IsInt("1230"))

	expect.True(IsInt("9123"))
	expect.True(IsInt("1923"))
	expect.True(IsInt("1293"))
	expect.True(IsInt("1239"))

	expect.True(IsInt("01239"))
	expect.True(IsInt("91230"))

}

func TestIsIntCorrectNeg(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.True(IsInt("-0"))
	expect.True(IsInt("-1"))
	expect.True(IsInt("-2"))
	expect.True(IsInt("-7"))
	expect.True(IsInt("-8"))
	expect.True(IsInt("-9"))

	expect.True(IsInt("-000"))
	expect.True(IsInt("-999"))

	expect.True(IsInt("-123"))

	expect.True(IsInt("-0123"))
	expect.True(IsInt("-1023"))
	expect.True(IsInt("-1203"))
	expect.True(IsInt("-1230"))

	expect.True(IsInt("-9123"))
	expect.True(IsInt("-1923"))
	expect.True(IsInt("-1293"))
	expect.True(IsInt("-1239"))

	expect.True(IsInt("-01239"))
	expect.True(IsInt("-91230"))
}

func TestIsIntIncorrect(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.False(IsInt("123-"))
	expect.False(IsInt("1-23"))
	expect.False(IsInt("a123"))
	expect.False(IsInt("a123a"))
	expect.False(IsInt("1.23"))
	expect.False(IsInt("a123a"))
}

type testStringer string

func (s testStringer) String() string {
	return string(s)
}

func TestJoinStringers(t *testing.T) {
	expect := ttesting.NewExpect(t)

	v := testStringer("Hello")
	arr1 := []fmt.Stringer{}
	arr2 := []fmt.Stringer{v}
	arr3 := []fmt.Stringer{v, v, v, v}

	expect.Equal("", JoinStringers(arr1, ","))
	expect.Equal("Hello", JoinStringers(arr2, ","))
	expect.Equal("Hello,Hello,Hello,Hello", JoinStringers(arr3, ","))
}

func TestTrimToNumber(t *testing.T) {
	expect := ttesting.NewExpect(t)

	test := "abc-10"
	result := TrimToNumber(test)
	expect.Equal("-10", result)

	test = "abc10"
	result = TrimToNumber(test)
	expect.Equal("10", result)

	test = "10def"
	result = TrimToNumber(test)
	expect.Equal("10", result)

	test = "abc-10def"
	result = TrimToNumber(test)
	expect.Equal("-10", result)

	test = "abc10def"
	result = TrimToNumber(test)
	expect.Equal("10", result)
}

func TestAtoI64(t *testing.T) {
	expect := ttesting.NewExpect(t)

	val, err := AtoI64("016")
	expect.NoError(err)
	expect.Equal(int64(016), val)

	val, err = AtoI64("0x16")
	expect.NoError(err)
	expect.Equal(int64(0x16), val)

	val, err = AtoI64("16")
	expect.NoError(err)
	expect.Equal(int64(16), val)

	val, err = AtoI64("")
	expect.NoError(err)
	expect.Equal(int64(0), val)

	val, err = AtoI64("0")
	expect.NoError(err)
	expect.Equal(int64(0), val)
}

func TestAtoU64(t *testing.T) {
	expect := ttesting.NewExpect(t)

	val, err := AtoU64("016")
	expect.NoError(err)
	expect.Equal(uint64(016), val)

	val, err = AtoU64("0x16")
	expect.NoError(err)
	expect.Equal(uint64(0x16), val)

	val, err = AtoU64("16")
	expect.NoError(err)
	expect.Equal(uint64(16), val)

	val, err = AtoU64("")
	expect.NoError(err)
	expect.Equal(uint64(0), val)

	val, err = AtoU64("0")
	expect.NoError(err)
	expect.Equal(uint64(0), val)
}
