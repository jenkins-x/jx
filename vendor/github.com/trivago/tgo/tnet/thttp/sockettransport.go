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

package thttp

import (
	"bufio"
	"net"
	"net/http"
	"time"
)

// SocketRoundtripper defines an http.RoundTripper that uses a unix domain socket
// as an endpoint.
type SocketRoundtripper struct {
	path string
}

// NewSocketTransport creates an http transport that listens to the "unix"
// protocol. I.e. requests must be of the form "unix://path".
func NewSocketTransport(path string) *http.Transport {
	socket := NewSocketRoundTripper(path)
	transport := &http.Transport{}
	transport.RegisterProtocol("unix", socket)
	return transport
}

// NewSocketRoundTripper creates a http.RoundTripper using a unix domain socket
func NewSocketRoundTripper(path string) http.RoundTripper {
	return SocketRoundtripper{
		path: path,
	}
}

// RoundTrip dials the unix domain socket, sends the request and processes the
// response.
func (s SocketRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		defer req.Body.Close()
	}
	socket, err := net.Dial("unix", s.path)
	if err != nil {
		return nil, err
	}
	defer socket.Close()

	socket.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := req.Write(socket); err != nil {
		return nil, err
	}

	socket.SetReadDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(socket)
	return http.ReadResponse(reader, req)
}
