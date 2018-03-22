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

func TestMin(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.Equal(0, MinI(1, 0))
	expect.Equal(0, MinI(0, 1))
	expect.Equal(-1, MinI(-1, 1))
	expect.Equal(-1, MinI(1, -1))

	expect.Equal(int64(0), MinInt64(1, 0))
	expect.Equal(int64(0), MinInt64(0, 1))
	expect.Equal(int64(-1), MinInt64(-1, 1))
	expect.Equal(int64(-1), MinInt64(1, -1))

	expect.Equal(uint64(0), MinUint64(1, 0))
	expect.Equal(uint64(0), MinUint64(0, 1))
}

func TestMax(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.Equal(1, MaxI(1, 0))
	expect.Equal(1, MaxI(0, 1))
	expect.Equal(1, MaxI(-1, 1))
	expect.Equal(1, MaxI(1, -1))

	expect.Equal(int64(1), MaxInt64(1, 0))
	expect.Equal(int64(1), MaxInt64(0, 1))
	expect.Equal(int64(1), MaxInt64(-1, 1))
	expect.Equal(int64(1), MaxInt64(1, -1))

	expect.Equal(uint64(1), MaxUint64(1, 0))
	expect.Equal(uint64(1), MaxUint64(0, 1))
}

func TestMin3(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.Equal(0, Min3I(0, 1, 2))
	expect.Equal(0, Min3I(0, 2, 1))
	expect.Equal(0, Min3I(2, 1, 0))
	expect.Equal(0, Min3I(2, 0, 1))
	expect.Equal(0, Min3I(1, 0, 2))
	expect.Equal(0, Min3I(1, 2, 0))

	expect.Equal(-1, Min3I(-1, 0, 1))
	expect.Equal(-1, Min3I(-1, 1, 0))
	expect.Equal(-1, Min3I(0, -1, 1))
	expect.Equal(-1, Min3I(0, 1, -1))
	expect.Equal(-1, Min3I(1, -1, 0))
	expect.Equal(-1, Min3I(1, 0, -1))

	expect.Equal(int64(0), Min3Int64(0, 1, 2))
	expect.Equal(int64(0), Min3Int64(0, 2, 1))
	expect.Equal(int64(0), Min3Int64(2, 1, 0))
	expect.Equal(int64(0), Min3Int64(2, 0, 1))
	expect.Equal(int64(0), Min3Int64(1, 0, 2))
	expect.Equal(int64(0), Min3Int64(1, 2, 0))

	expect.Equal(int64(-1), Min3Int64(-1, 0, 1))
	expect.Equal(int64(-1), Min3Int64(-1, 1, 0))
	expect.Equal(int64(-1), Min3Int64(0, -1, 1))
	expect.Equal(int64(-1), Min3Int64(0, 1, -1))
	expect.Equal(int64(-1), Min3Int64(1, -1, 0))
	expect.Equal(int64(-1), Min3Int64(1, 0, -1))

	expect.Equal(uint64(0), Min3Uint64(0, 1, 2))
	expect.Equal(uint64(0), Min3Uint64(0, 2, 1))
	expect.Equal(uint64(0), Min3Uint64(2, 1, 0))
	expect.Equal(uint64(0), Min3Uint64(2, 0, 1))
	expect.Equal(uint64(0), Min3Uint64(1, 0, 2))
	expect.Equal(uint64(0), Min3Uint64(1, 2, 0))
}

func TestMax3(t *testing.T) {
	expect := ttesting.NewExpect(t)

	expect.Equal(2, Max3I(0, 1, 2))
	expect.Equal(2, Max3I(0, 2, 1))
	expect.Equal(2, Max3I(2, 1, 0))
	expect.Equal(2, Max3I(2, 0, 1))
	expect.Equal(2, Max3I(1, 0, 2))
	expect.Equal(2, Max3I(1, 2, 0))

	expect.Equal(1, Max3I(-1, 0, 1))
	expect.Equal(1, Max3I(-1, 1, 0))
	expect.Equal(1, Max3I(0, -1, 1))
	expect.Equal(1, Max3I(0, 1, -1))
	expect.Equal(1, Max3I(1, -1, 0))
	expect.Equal(1, Max3I(1, 0, -1))

	expect.Equal(int64(2), Max3Int64(0, 1, 2))
	expect.Equal(int64(2), Max3Int64(0, 2, 1))
	expect.Equal(int64(2), Max3Int64(2, 1, 0))
	expect.Equal(int64(2), Max3Int64(2, 0, 1))
	expect.Equal(int64(2), Max3Int64(1, 0, 2))
	expect.Equal(int64(2), Max3Int64(1, 2, 0))

	expect.Equal(int64(1), Max3Int64(-1, 0, 1))
	expect.Equal(int64(1), Max3Int64(-1, 1, 0))
	expect.Equal(int64(1), Max3Int64(0, -1, 1))
	expect.Equal(int64(1), Max3Int64(0, 1, -1))
	expect.Equal(int64(1), Max3Int64(1, -1, 0))
	expect.Equal(int64(1), Max3Int64(1, 0, -1))

	expect.Equal(uint64(2), Max3Uint64(0, 1, 2))
	expect.Equal(uint64(2), Max3Uint64(0, 2, 1))
	expect.Equal(uint64(2), Max3Uint64(2, 1, 0))
	expect.Equal(uint64(2), Max3Uint64(2, 0, 1))
	expect.Equal(uint64(2), Max3Uint64(1, 0, 2))
	expect.Equal(uint64(2), Max3Uint64(1, 2, 0))
}
