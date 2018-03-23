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
	"io"
	"testing"
)

func TestByteWriter(t *testing.T) {
	expect := ttesting.NewExpect(t)
	buffer := make([]byte, 8)

	writer := NewByteWriter(&buffer)
	expect.Equal(8, len(buffer))

	writer.Reset()
	expect.Equal(0, len(buffer))

	size, err := writer.Write([]byte{1, 2, 3, 4, 5})
	expect.NoError(err)
	expect.Equal(5, size)
	expect.Equal(5, len(buffer))

	size, err = writer.Write([]byte{1, 2, 3, 4, 5})
	expect.Equal(io.EOF, err)
	expect.Equal(3, size)
	expect.Equal(8, len(buffer))
}
