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
	"bytes"
	"encoding/binary"
	"github.com/trivago/tgo/tstrings"
	"io"
)

// BufferedReaderFlags is an enum to configure a buffered reader
type BufferedReaderFlags byte

const (
	// BufferedReaderFlagDelimiter enables reading for a delimiter. This flag is
	// ignored if an MLE flag is set.
	BufferedReaderFlagDelimiter = BufferedReaderFlags(0)

	// BufferedReaderFlagMLE enables reading if length encoded messages.
	// Runlength is read as ASCII (to uint64) until the first byte (ASCII char)
	// of the delimiter string.
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE = BufferedReaderFlags(1)

	// BufferedReaderFlagMLE8 enables reading if length encoded messages.
	// Runlength is read as binary (to uint8).
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE8 = BufferedReaderFlags(2)

	// BufferedReaderFlagMLE16 enables reading if length encoded messages.
	// Runlength is read as binary (to uint16).
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE16 = BufferedReaderFlags(3)

	// BufferedReaderFlagMLE32 enables reading if length encoded messages.
	// Runlength is read as binary (to uint32).
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE32 = BufferedReaderFlags(4)

	// BufferedReaderFlagMLE64 enables reading if length encoded messages.
	// Runlength is read as binary (to uint64).
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE64 = BufferedReaderFlags(5)

	// BufferedReaderFlagMLEFixed enables reading messages with a fixed length.
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLEFixed = BufferedReaderFlags(6)

	// BufferedReaderFlagMaskMLE is a bitmask to mask out everything but MLE flags
	BufferedReaderFlagMaskMLE = BufferedReaderFlags(7)

	// BufferedReaderFlagBigEndian sets binary reading to big endian encoding.
	BufferedReaderFlagBigEndian = BufferedReaderFlags(8)

	// BufferedReaderFlagEverything will keep MLE and/or delimiters when
	// building a message.
	BufferedReaderFlagEverything = BufferedReaderFlags(16)
)

type bufferError string

func (b bufferError) Error() string {
	return string(b)
}

// BufferReadCallback defines the function signature for callbacks passed to
// ReadAll.
type BufferReadCallback func(msg []byte)

// BufferDataInvalid is returned when a parsing encounters an error
var BufferDataInvalid = bufferError("Invalid data")

// BufferedReader is a helper struct to read from any io.Reader into a byte
// slice. The data can arrive "in pieces" and will be assembled.
// A data "piece" is considered complete if a delimiter or a certain runlength
// has been reached. The latter has to be enabled by flag and will disable the
// default behavior, which is looking for a delimiter string.
type BufferedReader struct {
	data       []byte
	delimiter  []byte
	parse      func() ([]byte, int)
	paramMLE   int
	growSize   int
	end        int
	encoding   binary.ByteOrder
	flags      BufferedReaderFlags
	incomplete bool
}

// NewBufferedReader creates a new buffered reader that reads messages from a
// continuous stream of bytes.
// Messages can be separated from the stream by using common methods such as
// fixed size, encoded message length or delimiter string.
// The internal buffer is grown statically (by its original size) if necessary.
// bufferSize defines the initial size / grow size of the buffer
// flags configures the parsing method
// offsetOrLength sets either the runlength offset or fixed message size
// delimiter defines the delimiter used for textual message parsing
func NewBufferedReader(bufferSize int, flags BufferedReaderFlags, offsetOrLength int, delimiter string) *BufferedReader {
	buffer := BufferedReader{
		data:       make([]byte, bufferSize),
		delimiter:  []byte(delimiter),
		paramMLE:   offsetOrLength,
		encoding:   binary.LittleEndian,
		end:        0,
		flags:      flags,
		growSize:   bufferSize,
		incomplete: true,
	}

	if flags&BufferedReaderFlagBigEndian != 0 {
		buffer.encoding = binary.BigEndian
	}

	if flags&BufferedReaderFlagMaskMLE == 0 {
		buffer.parse = buffer.parseDelimiter
	} else {
		switch flags & BufferedReaderFlagMaskMLE {
		default:
			buffer.parse = buffer.parseMLEText
		case BufferedReaderFlagMLE8:
			buffer.parse = buffer.parseMLE8
		case BufferedReaderFlagMLE16:
			buffer.parse = buffer.parseMLE16
		case BufferedReaderFlagMLE32:
			buffer.parse = buffer.parseMLE32
		case BufferedReaderFlagMLE64:
			buffer.parse = buffer.parseMLE64
		case BufferedReaderFlagMLEFixed:
			buffer.parse = buffer.parseMLEFixed
		}
	}

	return &buffer
}

// Reset clears the buffer by resetting its internal state
func (buffer *BufferedReader) Reset(sequence uint64) {
	buffer.end = 0
	buffer.incomplete = true
}

// general message extraction part of all parser methods
func (buffer *BufferedReader) extractMessage(messageLen int, msgStartIdx int) ([]byte, int) {
	nextMsgIdx := msgStartIdx + messageLen
	if nextMsgIdx > buffer.end {
		return nil, 0 // ### return, incomplete ###
	}
	if buffer.flags&BufferedReaderFlagEverything != 0 {
		msgStartIdx = 0
	}
	return buffer.data[msgStartIdx:nextMsgIdx], nextMsgIdx
}

// messages have a fixed size
func (buffer *BufferedReader) parseMLEFixed() ([]byte, int) {
	return buffer.extractMessage(buffer.paramMLE, 0)
}

// messages are separated by a delimiter string
func (buffer *BufferedReader) parseDelimiter() ([]byte, int) {
	delimiterIdx := bytes.Index(buffer.data[:buffer.end], buffer.delimiter)
	if delimiterIdx == -1 {
		return nil, 0 // ### return, incomplete ###
	}

	messageLen := delimiterIdx
	if buffer.flags&BufferedReaderFlagEverything != 0 {
		messageLen += len(buffer.delimiter)
	}

	data, nextMsgIdx := buffer.extractMessage(messageLen, 0)

	if data != nil && buffer.flags&BufferedReaderFlagEverything == 0 {
		nextMsgIdx += len(buffer.delimiter)
	}
	return data, nextMsgIdx
}

// messages are separeated length encoded by ASCII number and (an optional)
// delimiter.
func (buffer *BufferedReader) parseMLEText() ([]byte, int) {
	messageLen, msgStartIdx := tstrings.Btoi(buffer.data[buffer.paramMLE:buffer.end])
	if msgStartIdx == 0 {
		return nil, -1 // ### return, malformed ###
	}

	msgStartIdx += buffer.paramMLE
	// Read delimiter if necessary (check if valid runlength)
	if delimiterLen := len(buffer.delimiter); delimiterLen > 0 {
		msgStartIdx += delimiterLen
		if msgStartIdx >= buffer.end {
			return nil, 0 // ### return, incomplete ###
		}
		if !bytes.Equal(buffer.data[msgStartIdx-delimiterLen:msgStartIdx], buffer.delimiter) {
			return nil, -1 // ### return, malformed ###
		}
	}

	return buffer.extractMessage(int(messageLen), msgStartIdx)
}

// messages are separated binary length encoded
func (buffer *BufferedReader) parseMLE8() ([]byte, int) {
	var messageLen uint8
	reader := bytes.NewReader(buffer.data[buffer.paramMLE:buffer.end])
	err := binary.Read(reader, buffer.encoding, &messageLen)
	if err != nil {
		return nil, -1 // ### return, malformed ###
	}
	return buffer.extractMessage(int(messageLen), buffer.paramMLE+1)
}

// messages are separated binary length encoded
func (buffer *BufferedReader) parseMLE16() ([]byte, int) {
	var messageLen uint16
	reader := bytes.NewReader(buffer.data[buffer.paramMLE:buffer.end])
	err := binary.Read(reader, buffer.encoding, &messageLen)
	if err != nil {
		return nil, -1 // ### return, malformed ###
	}
	return buffer.extractMessage(int(messageLen), buffer.paramMLE+2)
}

// messages are separated binary length encoded
func (buffer *BufferedReader) parseMLE32() ([]byte, int) {
	var messageLen uint32
	reader := bytes.NewReader(buffer.data[buffer.paramMLE:buffer.end])
	err := binary.Read(reader, buffer.encoding, &messageLen)
	if err != nil {
		return nil, -1 // ### return, malformed ###
	}
	return buffer.extractMessage(int(messageLen), buffer.paramMLE+4)
}

// messages are separated binary length encoded
func (buffer *BufferedReader) parseMLE64() ([]byte, int) {
	var messageLen uint64
	reader := bytes.NewReader(buffer.data[buffer.paramMLE:buffer.end])
	err := binary.Read(reader, buffer.encoding, &messageLen)
	if err != nil {
		return nil, -1 // ### return, malformed ###
	}
	return buffer.extractMessage(int(messageLen), buffer.paramMLE+8)
}

// ReadAll calls ReadOne as long as there are messages in the stream.
// Messages will be send to the given write callback.
// If callback is nil, data will be read and discarded.
func (buffer *BufferedReader) ReadAll(reader io.Reader, callback BufferReadCallback) error {
	for {
		data, more, err := buffer.ReadOne(reader)
		if data != nil && callback != nil {
			callback(data)
		}

		if err != nil {
			return err // ### return, error ###
		}

		if !more {
			return nil // ### return, done ###
		}
	}
}

// ReadOne reads the next message from the given stream (if possible) and
// generates a sequence number for this message.
// The more return parameter is set to true if there are still messages or parts
// of messages in the stream. Data and seq is only set if a complete message
// could be parsed.
// Errors are returned if reading from the stream failed or the parser
// encountered an error.
func (buffer *BufferedReader) ReadOne(reader io.Reader) (data []byte, more bool, err error) {
	if buffer.incomplete {
		bytesRead, err := reader.Read(buffer.data[buffer.end:])

		if err != nil && err != io.EOF {
			return nil, buffer.end > 0, err // ### return, error reading ###
		}

		if bytesRead == 0 {
			return nil, buffer.end > 0, err // ### return, no data ###
		}

		buffer.end += bytesRead
		buffer.incomplete = false
	}

	msgData, nextMsgIdx := buffer.parse()

	if nextMsgIdx == -1 {
		buffer.end = 0
		buffer.incomplete = true
		return nil, true, BufferDataInvalid // ### return, invalid data ###
	}

	if msgData == nil {
		// Check if buffer needs to be resized
		if len(buffer.data) == buffer.end {
			temp := buffer.data
			buffer.data = make([]byte, len(buffer.data)+buffer.growSize)
			copy(buffer.data, temp)
		}
		buffer.incomplete = true
		return nil, true, err // ### return, incomplete ###
	}

	msgDataCopy := make([]byte, len(msgData))
	copy(msgDataCopy, msgData)

	if nextMsgIdx < buffer.end {
		copy(buffer.data, buffer.data[nextMsgIdx:buffer.end])
		buffer.end -= nextMsgIdx
	} else {
		buffer.end = 0
		buffer.incomplete = true
	}

	return msgDataCopy, buffer.end > 0, err
}
