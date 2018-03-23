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
	"net"
)

// StopListener is a wrapper around net.TCPListener that allows to gracefully
// shut down a connection loop.
type StopListener struct {
	*net.TCPListener
	active bool
}

// StopRequestError is not an error but a note that the connection was requested
// to be closed.
type StopRequestError struct{}

// NewStopListener creates a new, stoppable TCP server connection.
// Address needs to be cmpliant to net.Listen.
func NewStopListener(address string) (*StopListener, error) {
	listen, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err // ### return, could not connect ###
	}

	return &StopListener{
		TCPListener: listen.(*net.TCPListener),
		active:      true,
	}, nil
}

// Error implements the standard error interface
func (err StopRequestError) Error() string {
	return "Connection stop request"
}

// Accept is analogous to net.TCPListener.Accept but may return a connection
// closed error if the connection was requested to shut down.
func (listen *StopListener) Accept() (net.Conn, error) {
	conn, err := listen.TCPListener.Accept()
	if !listen.active {
		return nil, StopRequestError{} // ### return, stop requested ###
	}
	if err != nil {
		return nil, err // ### return, error ###
	}

	return conn, err
}

// Close requests a connection close on this listener
func (listen *StopListener) Close() error {
	listen.active = false
	return listen.TCPListener.Close()
}
