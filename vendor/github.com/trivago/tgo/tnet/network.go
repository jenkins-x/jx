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
	"github.com/trivago/tgo/tstrings"
	"io"
	"net"
	"strconv"
	"strings"
	"syscall"
)

// URI stores parts parsed from an URI string.
type URI struct {
	Protocol string
	Address  string
	Port     uint16
}

// ParseAddress takes an address and tries to extract the protocol from it.
// Protocols may be prepended by the "protocol://" notation.
// If no protocol is given, defaultProtocol is returned.
// The first parameter returned is the address, the second denotes the protocol.
// The protocol is allways returned as lowercase string. If only a number is
// given as an address the number is assumed to be a port. As of this "80" and
// ":80" will return ":80" as the address.
func ParseAddress(addressString string, defaultProtocol string) (protocol, address string) {
	protocolIdx := strings.Index(addressString, "://")
	if protocolIdx == -1 {
		protocol = strings.ToLower(defaultProtocol)
		address = addressString
	} else {
		protocol = strings.ToLower(addressString[:protocolIdx])
		address = addressString[protocolIdx+3:]
	}

	// Allow sole numbers as a synonym for ":port"
	if tstrings.IsInt(address) {
		address = ":" + address
	}

	return
}

// SplitAddress splits an address of the form "protocol://host:port" into its
// components. If no protocol is given, defaultProtocol is used.
// This function uses net.SplitHostPort and ParseAddress.
func SplitAddress(addressString string, defaultProtocol string) (protocol, host, port string, err error) {
	protocol, host = ParseAddress(addressString, defaultProtocol)
	if protocol == "unix" || protocol == "unixgram" || protocol == "unixpacket" {
		return
	}

	host, port, err = net.SplitHostPort(host)
	return
}

// SplitAddressToURI acts like SplitAddress but returns an URI struct instead.
func SplitAddressToURI(addressString string, defaultProtocol string) (uri URI, err error) {
	portString := ""
	uri.Protocol, uri.Address, portString, err = SplitAddress(addressString, defaultProtocol)
	portNum, _ := strconv.ParseInt(portString, 10, 16)
	uri.Port = uint16(portNum)
	return uri, err
}

// IsDisconnectedError returns true if the given error is related to a
// disconnected socket.
func IsDisconnectedError(err error) bool {
	if err == io.EOF {
		return true // ### return, closed stream ###
	}

	netErr, isNetErr := err.(*net.OpError)
	if isNetErr {
		errno, isErrno := netErr.Err.(syscall.Errno)
		if isErrno {
			switch errno {
			default:
			case syscall.ECONNRESET:
				return true // ### return, close connection ###
			}
		}
	}

	return false
}
