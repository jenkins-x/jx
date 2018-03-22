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

package tmath

import (
	"github.com/trivago/tgo/ttesting"
	"testing"
)

func TestNextPowerOf2U64(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.Equal(uint64(0), NextPowerOf2U64(uint64(0)))
	expect.Equal(uint64(1), NextPowerOf2U64(uint64(1)))
	expect.Equal(uint64(2), NextPowerOf2U64(uint64(2)))
	expect.Equal(uint64(4), NextPowerOf2U64(uint64(4)))
	expect.Equal(uint64(8), NextPowerOf2U64(uint64(8)))
	expect.Equal(uint64(16), NextPowerOf2U64(uint64(16)))

	expect.Equal(uint64(4), NextPowerOf2U64(uint64(3)))
	expect.Equal(uint64(8), NextPowerOf2U64(uint64(7)))
	expect.Equal(uint64(16), NextPowerOf2U64(uint64(15)))
}

func TestNextPowerOf2U32(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.Equal(uint32(0), NextPowerOf2U32(uint32(0)))
	expect.Equal(uint32(1), NextPowerOf2U32(uint32(1)))
	expect.Equal(uint32(2), NextPowerOf2U32(uint32(2)))
	expect.Equal(uint32(4), NextPowerOf2U32(uint32(4)))
	expect.Equal(uint32(8), NextPowerOf2U32(uint32(8)))
	expect.Equal(uint32(16), NextPowerOf2U32(uint32(16)))

	expect.Equal(uint32(4), NextPowerOf2U32(uint32(3)))
	expect.Equal(uint32(8), NextPowerOf2U32(uint32(7)))
	expect.Equal(uint32(16), NextPowerOf2U32(uint32(15)))
}
func TestNextPowerOf2U16(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.Equal(uint16(0), NextPowerOf2U16(uint16(0)))
	expect.Equal(uint16(1), NextPowerOf2U16(uint16(1)))
	expect.Equal(uint16(2), NextPowerOf2U16(uint16(2)))
	expect.Equal(uint16(4), NextPowerOf2U16(uint16(4)))
	expect.Equal(uint16(8), NextPowerOf2U16(uint16(8)))
	expect.Equal(uint16(16), NextPowerOf2U16(uint16(16)))

	expect.Equal(uint16(4), NextPowerOf2U16(uint16(3)))
	expect.Equal(uint16(8), NextPowerOf2U16(uint16(7)))
	expect.Equal(uint16(16), NextPowerOf2U16(uint16(15)))
}
