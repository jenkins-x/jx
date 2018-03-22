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
	"github.com/trivago/tgo/ttesting"
	"testing"
)

func TestByteWriteByte(t *testing.T) {
	expect := ttesting.NewExpect(t)
	stream := NewByteStream(1)

	data := []byte("a")
	stream.WriteByte(data[0])
	expect.Equal("a", stream.String())
}

func TestByteStream(t *testing.T) {
	expect := ttesting.NewExpect(t)

	stream := NewByteStream(1)
	expect.Equal(1, stream.Cap())
	expect.Equal(0, stream.Len())
	expect.Equal(stream.Len(), len(stream.Bytes()))
	expect.Equal(stream.Cap(), cap(stream.Bytes()))

	stream.Write([]byte("a"))
	expect.Equal(1, stream.Cap())
	expect.Equal(1, stream.Len())
	expect.Equal(stream.Len(), len(stream.Bytes()))
	expect.Equal(stream.Cap(), cap(stream.Bytes()))

	stream.Write([]byte("bc"))
	expect.Equal("abc", string(stream.Bytes()))
	expect.Equal(3, stream.Cap())
	expect.Equal(3, stream.Len())
	expect.Equal(stream.Len(), len(stream.Bytes()))
	expect.Equal(stream.Cap(), cap(stream.Bytes()))

	stream.Reset()
	expect.Equal("", string(stream.Bytes()))
	expect.Equal(3, stream.Cap())
	expect.Equal(0, stream.Len())
	expect.Equal(stream.Len(), len(stream.Bytes()))
	expect.Equal(stream.Cap(), cap(stream.Bytes()))

	stream.Write([]byte("bc"))
	expect.Equal("bc", string(stream.Bytes()))
	expect.Equal(3, stream.Cap())
	expect.Equal(2, stream.Len())
	expect.Equal(stream.Len(), len(stream.Bytes()))
	expect.Equal(stream.Cap(), cap(stream.Bytes()))
}
