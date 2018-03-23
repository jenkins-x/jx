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

func TestStopListenerNewStopListener(t *testing.T) {
	expect := ttesting.NewExpect(t)

	listener, err := NewStopListener("incompliantAddress")
	expect.Nil(listener)
	expect.NotNil(err)

	listener, err = NewStopListener("localhost:8080")
	expect.Nil(err)
	expect.NotNil(listener)
	err = listener.Close()
	expect.Nil(err)
}

func TestStopListenerAccept(t *testing.T) {

}
