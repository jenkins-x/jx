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
	"io"
	"sync"
)

type logCache struct {
	flushing sync.Mutex
	messages [][]byte
}

// Write caches the passed message. Blocked by flush function.
func (log *logCache) Write(message []byte) (int, error) {
	log.flushing.Lock()
	defer log.flushing.Unlock()

	if len(message) > 0 {
		messageCopy := make([]byte, len(message))
		copy(messageCopy, message)
		log.messages = append(log.messages, messageCopy)
	}
	return len(message), nil
}

// Flush writes all messages to the given writer and clears the list
// of stored messages. Blocks Write function.
func (log *logCache) Flush(writer io.Writer) {
	log.flushing.Lock()
	defer log.flushing.Unlock()

	for _, message := range log.messages {
		writer.Write(message)
	}
	log.messages = [][]byte{}
}
