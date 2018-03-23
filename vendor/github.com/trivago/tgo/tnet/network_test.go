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

package tnet

import (
	"github.com/trivago/tgo/ttesting"
	"testing"
)

func TestParseAddress(t *testing.T) {
	expect := ttesting.NewExpect(t)

	proto, addr := ParseAddress("unix://test.socket", "")
	expect.Equal("unix", proto)
	expect.Equal("test.socket", addr)

	proto, addr = ParseAddress("udp://192.168.0.1:20", "")
	expect.Equal("udp", proto)
	expect.Equal("192.168.0.1:20", addr)

	proto, addr = ParseAddress("192.168.0.1:20", "tcp")
	expect.Equal("tcp", proto)
	expect.Equal("192.168.0.1:20", addr)

	proto, addr = ParseAddress("80", "tcp")
	expect.Equal("tcp", proto)
	expect.Equal(":80", addr)

	proto, addr = ParseAddress("tcp://80", "")
	expect.Equal("tcp", proto)
	expect.Equal(":80", addr)
}

func TestSplitAddress(t *testing.T) {
	expect := ttesting.NewExpect(t)

	proto, addr, port, err := SplitAddress("unix://test.socket", "")
	expect.NoError(err)
	expect.Equal("unix", proto)
	expect.Equal("test.socket", addr)
	expect.Equal("", port)

	proto, addr, port, err = SplitAddress("udp://192.168.0.1:20", "")
	expect.NoError(err)
	expect.Equal("udp", proto)
	expect.Equal("192.168.0.1", addr)
	expect.Equal("20", port)

	proto, addr, port, err = SplitAddress("192.168.0.1:20", "tcp")
	expect.NoError(err)
	expect.Equal("tcp", proto)
	expect.Equal("192.168.0.1", addr)
	expect.Equal("20", port)

	proto, addr, port, err = SplitAddress("80", "tcp")
	expect.NoError(err)
	expect.Equal("tcp", proto)
	expect.Equal("", addr)
	expect.Equal("80", port)

	proto, addr, port, err = SplitAddress("tcp://80", "")
	expect.NoError(err)
	expect.Equal("tcp", proto)
	expect.Equal("", addr)
	expect.Equal("80", port)
}

func TestSplitAddressToURI(t *testing.T) {
	expect := ttesting.NewExpect(t)

	uri, err := SplitAddressToURI("unix://test.socket", "")
	expect.NoError(err)
	expect.Equal("unix", uri.Protocol)
	expect.Equal("test.socket", uri.Address)
	expect.Equal(uint16(0), uri.Port)

	uri, err = SplitAddressToURI("udp://192.168.0.1:20", "")
	expect.NoError(err)
	expect.Equal("udp", uri.Protocol)
	expect.Equal("192.168.0.1", uri.Address)
	expect.Equal(uint16(20), uri.Port)

	uri, err = SplitAddressToURI("192.168.0.1:20", "tcp")
	expect.NoError(err)
	expect.Equal("tcp", uri.Protocol)
	expect.Equal("192.168.0.1", uri.Address)
	expect.Equal(uint16(20), uri.Port)

	uri, err = SplitAddressToURI("80", "tcp")
	expect.NoError(err)
	expect.Equal("tcp", uri.Protocol)
	expect.Equal("", uri.Address)
	expect.Equal(uint16(80), uri.Port)

	uri, err = SplitAddressToURI("tcp://80", "")
	expect.NoError(err)
	expect.Equal("tcp", uri.Protocol)
	expect.Equal("", uri.Address)
	expect.Equal(uint16(80), uri.Port)
}
