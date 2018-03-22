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

package tgo

import (
	"log"
	"net"
	"time"
)

// MetricServer contains state information about the metric server process
type MetricServer struct {
	metrics *Metrics
	running bool
	listen  net.Listener
	updates *time.Ticker
}

// NewMetricServer creates a new server state for a metric server based on
// the global Metric variable.
func NewMetricServer() *MetricServer {
	return &MetricServer{
		metrics: Metric,
		running: false,
		updates: time.NewTicker(time.Second),
	}
}

// NewMetricServerFor creates a new server state for a metric server based
// on a custom Metrics variable
func NewMetricServerFor(m *Metrics) *MetricServer {
	return &MetricServer{
		metrics: m,
		running: false,
		updates: time.NewTicker(time.Second),
	}
}

func (server *MetricServer) handleMetricRequest(conn net.Conn) {
	defer conn.Close()

	data, err := server.metrics.Dump()
	if err != nil {
		conn.Write([]byte(err.Error()))
	} else {
		conn.Write(data)
	}
	conn.Write([]byte{'\n'})
	conn.Close()
}

func (server *MetricServer) sysUpdate() {
	for server.running {
		_, running := <-server.updates.C
		if !running {
			return // ### return, timer has been stopped ###
		}

		server.metrics.UpdateSystemMetrics()
	}
}

// Start causes a metric server to listen for a specific address and port.
// If this address/port is accessed a JSON containing all metrics will be
// returned and the connection is closed.
// You can use the standard go notation for addresses like ":80".
func (server *MetricServer) Start(address string) {
	if server.running {
		return
	}

	var err error
	server.listen, err = net.Listen("tcp", address)
	if err != nil {
		log.Print("Metrics: ", err)
		time.AfterFunc(time.Second*5, func() { server.Start(address) })
		return
	}

	server.running = true
	go server.sysUpdate()

	for server.running {
		client, err := server.listen.Accept()
		if err != nil {
			if server.running {
				log.Print("Metrics: ", err)
			}
			return // ### break ###
		}

		go server.handleMetricRequest(client)
	}
}

// Stop notifies the metric server to halt.
func (server *MetricServer) Stop() {
	server.running = false
	server.updates.Stop()
	if server.listen != nil {
		if err := server.listen.Close(); err != nil {
			log.Print("Metrics: ", err)
		}
	}
}
