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

import "io"

// ByteStream is a more lightweight variant of bytes.Buffer.
// The managed byte array is increased to the exact required size and never
// shrinks. Writing moves an internal offset (for appends) but reading always
// starts at offset 0.
type ByteStream struct {
	data        []byte
	writeOffset int
	readOffset  int
}

// NewByteStream creates a new byte stream of the desired capacity
func NewByteStream(capacity int) ByteStream {
	return ByteStream{
		data:        make([]byte, capacity),
		writeOffset: 0,
	}
}

// NewByteStreamFrom creates a new byte stream that starts with the given
// byte array.
func NewByteStreamFrom(data []byte) ByteStream {
	return ByteStream{
		data:        data,
		writeOffset: len(data),
	}
}

// SetCapacity assures that capacity bytes are available in the buffer, growing
// the managed byte array if needed. The buffer will grow by at least 64 bytes
// if growing is required.
func (stream *ByteStream) SetCapacity(capacity int) {
	if stream.Cap() < capacity {
		current := stream.data
		stream.data = make([]byte, capacity)
		if stream.writeOffset > 0 {
			copy(stream.data, current[:stream.writeOffset])
		}
	}
}

// Reset sets the internal write offset to 0 and calls ResetRead
func (stream *ByteStream) Reset() {
	stream.writeOffset = 0
	stream.ResetRead()
}

// ResetRead sets the internal read offset to 0. The read offset is only used
// within the Read function to assure io.Reader compatibility
func (stream *ByteStream) ResetRead() {
	stream.readOffset = 0
}

// Len returns the length of the underlying array.
// This is equal to len(stream.Bytes()).
func (stream ByteStream) Len() int {
	return stream.writeOffset
}

// Cap returns the capacity of the underlying array.
// This is equal to cap(stream.Bytes()).
func (stream ByteStream) Cap() int {
	return len(stream.data)
}

// Bytes returns a slice of the underlying byte array containing all written
// data up to this point.
func (stream ByteStream) Bytes() []byte {
	return stream.data[:stream.writeOffset]
}

// String returns a string of the underlying byte array containing all written
// data up to this point.
func (stream ByteStream) String() string {
	return string(stream.data[:stream.writeOffset])
}

// Write implements the io.Writer interface.
// This function assures that the capacity of the underlying byte array is
// enough to store the incoming amount of data. Subsequent writes will allways
// append to the end of the stream until Reset() is called.
func (stream *ByteStream) Write(source []byte) (int, error) {
	sourceLen := len(source)
	if sourceLen == 0 {
		return 0, nil
	}

	stream.SetCapacity(stream.writeOffset + sourceLen)
	copy(stream.data[stream.writeOffset:], source[:sourceLen])
	stream.writeOffset += sourceLen

	return sourceLen, nil
}

// WriteString is a convenience wrapper for Write([]byte(source))
func (stream *ByteStream) WriteString(source string) (int, error) {
	return stream.Write([]byte(source))
}

// WriteByte writes a single byte to the stream. Capacity will be ensured.
func (stream *ByteStream) WriteByte(source byte) error {
	stream.SetCapacity(stream.writeOffset + 1)
	stream.data[stream.writeOffset] = source
	stream.writeOffset++
	return nil
}

// Read implements the io.Reader interface.
// The underlying array is copied to target and io.EOF is returned once no data
// is left to be read. Please note that ResetRead can rewind the internal read
// index for this operation.
func (stream *ByteStream) Read(target []byte) (int, error) {
	if stream.readOffset >= stream.writeOffset {
		return 0, io.EOF // ### return, no new data to read ###
	}

	bytesCopied := copy(target, stream.data[stream.readOffset:stream.writeOffset])
	stream.readOffset += bytesCopied

	if stream.readOffset >= stream.writeOffset {
		return bytesCopied, io.EOF
	}
	return bytesCopied, nil
}
