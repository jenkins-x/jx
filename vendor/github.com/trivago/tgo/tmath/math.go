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

// MaxI returns the maximum out of two integers
func MaxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MaxInt64 returns the maximum out of two signed 64bit integers
func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// MaxUint64 returns the maximum out of two unsigned 64bit integers
func MaxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// Max3I returns the maximum out of three integers
func Max3I(a, b, c int) int {
	max := a
	if b > max {
		max = b
	}
	if c > max {
		max = c
	}
	return max
}

// Max3Int64 returns the maximum out of three signed 64-bit integers
func Max3Int64(a, b, c int64) int64 {
	max := a
	if b > max {
		max = b
	}
	if c > max {
		max = c
	}
	return max
}

// Max3Uint64 returns the maximum out of three unsigned 64-bit integers
func Max3Uint64(a, b, c uint64) uint64 {
	max := a
	if b > max {
		max = b
	}
	if c > max {
		max = c
	}
	return max
}

// MinI returns the minimum out of two integers
func MinI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MinInt64 returns the minimum out of two signed 64-bit integers
func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// MinUint64 returns the minimum out of two unsigned 64-bit integers
func MinUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Min3I returns the minimum out of three integers
func Min3I(a, b, c int) int {
	min := a
	if b < min {
		min = b
	}
	if c < min {
		min = c
	}
	return min
}

// Min3Int64 returns the minimum out of three signed 64-bit integers
func Min3Int64(a, b, c int64) int64 {
	min := a
	if b < min {
		min = b
	}
	if c < min {
		min = c
	}
	return min
}

// Min3Uint64 returns the minimum out of three unsigned 64-bit integers
func Min3Uint64(a, b, c uint64) uint64 {
	min := a
	if b < min {
		min = b
	}
	if c < min {
		min = c
	}
	return min
}
