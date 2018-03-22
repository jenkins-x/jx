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

package tlog

import (
	"log"
)

type logLogger struct {
	logger *log.Logger
}

// Write sends the message to the io.Writer passed to Configure
func (lg logLogger) Write(message []byte) (int, error) {
	length := len(message)
	if length > 0 {
		lg.logger.Output(2, string(message))
	}
	return length, nil
}
