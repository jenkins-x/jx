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
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

var simpleEscapeChars = strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t", "\\f", "\f", "\\b", "\b")
var jsonEscapeChars = strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\\n", "\n", "\\r", "\r", "\\t", "\t", "\\f", "\f", "\\b", "\b")

// Unescape replaces occurrences of \\n, \\r and \\t with real escape codes.
func Unescape(text string) string {
	return simpleEscapeChars.Replace(text)
}

// EscapeJSON replaces occurrences of \ and " with escaped versions.
func EscapeJSON(text string) string {
	return jsonEscapeChars.Replace(text)
}

// ItoLen returns the length of an unsingned integer when converted to a string
func ItoLen(number uint64) int {
	switch {
	case number < 10:
		return 1
	default:
		return int(math.Log10(float64(number)) + 1)
	}
}

// Itob writes an unsigned integer to the start of a given byte buffer.
func Itob(number uint64, buffer []byte) error {
	numberLen := ItoLen(number)
	bufferLen := len(buffer)

	if numberLen > bufferLen {
		return fmt.Errorf("Number too large for buffer")
	}

	for i := numberLen - 1; i >= 0; i-- {
		buffer[i] = '0' + byte(number%10)
		number /= 10
	}

	return nil
}

// Itobe writes an unsigned integer to the end of a given byte buffer.
func Itobe(number uint64, buffer []byte) error {
	for i := len(buffer) - 1; i >= 0; i-- {
		buffer[i] = '0' + byte(number%10)
		number /= 10

		// Check here because the number 0 has to be written, too
		if number == 0 {
			return nil
		}
	}

	return fmt.Errorf("Number too large for buffer")
}

// Btoi is a fast byte buffer to unsigned int parser that reads until the first
// non-number character is found. It returns the number value as well as the
// length of the number string encountered.
// If a number could not be parsed the returned length will be 0
func Btoi(buffer []byte) (uint64, int) {
	number := uint64(0)
	index := 0
	bufferLen := len(buffer)

	for index < bufferLen && buffer[index] >= '0' && buffer[index] <= '9' {
		parsed := uint64(buffer[index] - '0')
		number = number*10 + parsed
		index++
	}

	return number, index
}

// IndexN returns the nth occurrences of sep in s or -1 if there is no nth
// occurrence of sep.
func IndexN(s, sep string, n int) int {
	sepIdx := 0
	for i := 0; i < n; i++ {
		nextIdx := strings.Index(s[sepIdx:], sep)
		if nextIdx == -1 {
			return -1 // ### return, not found ###
		}
		sepIdx += nextIdx + 1
	}
	return sepIdx - 1
}

// LastIndexN returns the nth occurrence of sep in s or -1 if there is no nth
// occurrence of sep. Searching is going from the end of the string to the start.
func LastIndexN(s, sep string, n int) int {
	if n == 0 {
		return -1 // ### return, nonsense ###
	}
	sepIdx := len(s)
	for i := 0; i < n; i++ {
		sepIdx = strings.LastIndex(s[:sepIdx], sep)
		if sepIdx == -1 {
			return -1 // ### return, not found ###
		}
	}
	return sepIdx
}

// IsInt returns true if the given string can be converted to an integer.
// The string must contain no whitespaces.
func IsInt(s string) bool {
	for i, c := range s {
		if (c < '0' || c > '9') && (c != '-' || i != 0) {
			return false
		}
	}
	return true
}

// IsJSON returns true if the given byte slice contains valid JSON data.
// You can access the results by utilizing the RawMessage returned.
func IsJSON(data []byte) (bool, error, *json.RawMessage) {
	delayedResult := new(json.RawMessage)
	err := json.Unmarshal(data, delayedResult)
	return err == nil, err, delayedResult

}

// ByteRef is a standard byte slice that is referencing a string.
// This type is considered unsafe and should only be used for fast conversion
// in case of read-only string access via []byte.
type ByteRef []byte

// NewByteRef creates an unsafe byte slice reference to the given string.
// The referenced data is not guaranteed to stay valid if no reference to the
// original string is held anymore. The returned ByteRef does not count as
// reference.
// Writing to the returned ByteRef will result in a segfault.
func NewByteRef(a string) ByteRef {
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&a))
	sliceHeader := &reflect.SliceHeader{
		Data: stringHeader.Data,
		Len:  stringHeader.Len,
		Cap:  stringHeader.Len,
	}
	return *(*ByteRef)(unsafe.Pointer(sliceHeader))
}

// StringRef is a standard string that is referencing the data of a byte slice.
// The string changes if the referenced byte slice changes. If the referenced
// byte slice changes size this data might get invalid and result in segfaults.
// As of that, this type is to be considered unsafe and should only be used
// in special, performance critical situations.
type StringRef string

// NewStringRef creates a unsafe string reference to to the given byte slice.
// The referenced data is not guaranteed to stay valid if no reference to the
// original byte slice is held anymore. The returned StringRef does not count as
// reference.
// Changing the size of the underlying byte slice will result in a segfault.
func NewStringRef(a []byte) StringRef {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&a))
	stringHeader := &reflect.StringHeader{
		Data: sliceHeader.Data,
		Len:  sliceHeader.Len,
	}
	return *(*StringRef)(unsafe.Pointer(stringHeader))
}

// JoinStringers works like strings.Join but takes an array of fmt.Stringer.
func JoinStringers(a []fmt.Stringer, sep string) string {
	if len(a) == 0 {
		return ""
	}
	if len(a) == 1 {
		return a[0].String()
	}

	str := make([]string, len(a))
	for i, s := range a {
		str[i] = s.String()
	}

	return strings.Join(str, sep)
}

// TrimToNumber removes all characters from the left and right of the string
// that are not in the set [\-0-9] on the left or [0-9] on the right
func TrimToNumber(text string) string {
	leftTrimmed := strings.TrimLeftFunc(text, func(r rune) bool {
		return (r < '0' || r > '9') && r != '-'
	})

	return strings.TrimRightFunc(leftTrimmed, func(r rune) bool {
		return r < '0' || r > '9'
	})
}

// GetNumBase scans the given string for its numeric base. If num starts with
// '0x' base 16 is assumed. Just '0' assumes base 8.
func GetNumBase(num string) (string, int) {
	switch {
	case len(num) == 0:
		return num, 10

	case len(num) > 1 && num[0] == '0':
		if num[1] == 'x' {
			return num[2:], 16
		}
		return num[1:], 8
	}
	return num, 10
}

// AtoI64 converts a numeric string to an int64, using GetNumBase to detect
// the numeric base for conversion.
func AtoI64(num string) (int64, error) {
	if len(num) == 0 {
		return 0, nil
	}
	n, b := GetNumBase(num)
	return strconv.ParseInt(n, b, 64)
}

// AtoU64 converts a numeric string to an uint64, using GetNumBase to detect
// the numeric base for conversion.
func AtoU64(num string) (uint64, error) {
	if len(num) == 0 {
		return 0, nil
	}
	n, b := GetNumBase(num)
	return strconv.ParseUint(n, b, 64)
}
