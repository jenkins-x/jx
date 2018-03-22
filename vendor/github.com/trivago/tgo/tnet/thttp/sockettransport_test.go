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
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/ttesting"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
)

const sockFile = "/tmp/tgo_http_socket_test.socket"

type mockHandler struct {
	expect ttesting.Expect
}

func (h mockHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	if h.expect.NotNil(req) && h.expect.NotNil(req.URL) {
		h.expect.Equal(sockFile, req.URL.String())
	}

	writer.Write([]byte("OK"))
}

func TestSocketTransport(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// Create endpoint

	if tio.FileExists(sockFile) {
		os.Remove(sockFile)
	}

	socket, err := net.Listen("unix", sockFile)
	expect.NoError(err)
	handler := mockHandler{
		expect: expect,
	}
	defer socket.Close()
	go http.Serve(socket, handler)

	// Send data

	transport := NewSocketTransport(sockFile)
	client := &http.Client{Transport: transport}

	rsp, err := client.Get("unix://" + sockFile)
	expect.NoError(err)
	expect.Equal(200, rsp.StatusCode)

	body, err := ioutil.ReadAll(rsp.Body)
	expect.NoError(err)
	expect.Equal("OK", string(body))
}
