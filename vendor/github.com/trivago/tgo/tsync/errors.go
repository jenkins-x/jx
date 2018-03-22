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

package tsync

import (
	"github.com/trivago/tgo/terrors"
)

// LockedError is returned when an item has been encountered as locked
type LockedError terrors.SimpleError

func (err LockedError) Error() string {
	return err.Error()
}

// TimeoutError is returned when a function returned because of a timeout
type TimeoutError terrors.SimpleError

func (err TimeoutError) Error() string {
	return err.Error()
}

// LimitError is returned when a datastructure reached its limit
type LimitError terrors.SimpleError

func (err LimitError) Error() string {
	return err.Error()
}
