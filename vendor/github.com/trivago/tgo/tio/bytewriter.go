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

package tio

import (
	"io"
)

// ByteWriter wraps a slice so that it can be passed to io.Writer compatible
// functions. The referenced slice will be resized.
type ByteWriter struct {
	buffer *[]byte
}

// NewByteWriter creates a new writer on the referenced byte slice.
func NewByteWriter(buffer *[]byte) ByteWriter {
	return ByteWriter{
		buffer: buffer,
	}
}

// Reset sets the length of the slize to 0.
func (b ByteWriter) Reset() {
	*b.buffer = (*b.buffer)[:0]
}

// Write writes data to the wrapped slice if there is enough space available.
// If not, data is written until the wrapped slice has reached its capacity
// and io.EOF is returned.
func (b ByteWriter) Write(data []byte) (int, error) {
	start := len(*b.buffer)
	size := len(data)
	capacity := cap(*b.buffer)

	if start+size > capacity {
		size := capacity - start
		*b.buffer = (*b.buffer)[:capacity]
		copy((*b.buffer)[start:], data[:size])
		return size, io.EOF
	}

	*b.buffer = (*b.buffer)[:start+size]
	copy((*b.buffer)[start:], data)

	return size, nil
}
