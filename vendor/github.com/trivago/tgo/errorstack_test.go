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
	"fmt"
	"github.com/trivago/tgo/ttesting"
	"testing"
)

type mockError struct {
}

func (e mockError) Error() string {
	return "test"
}

func TestPush(t *testing.T) {
	expect := ttesting.NewExpect(t)
	stack := NewErrorStack()

	var err mockError

	stack.Push(err)
	expect.Equal(err, stack.Top())

	stack.Pushf("test %d", 123)
	expect.Equal("test 123", stack.Top().Error())

	stack.PushAndDescribe("this is a", err)
	expect.Equal("this is a test", stack.Top().Error())
}

func TestPop(t *testing.T) {
	expect := ttesting.NewExpect(t)
	stack := NewErrorStack()

	var err mockError

	stack.Push(err)
	expect.Equal(len(stack.Errors()), 1)
	expect.NotNil(stack.OrNil())

	stack.Clear()
	expect.Equal(len(stack.Errors()), 0)
	expect.Nil(stack.OrNil())

	stack.Push(err)
	expect.Greater(len(stack.Errors()), 0)

	err2 := stack.Pop()
	expect.Equal(err2, err)
	expect.Equal(len(stack.Errors()), 0)
}

func TestFormat(t *testing.T) {
	expect := ttesting.NewExpect(t)
	stack := NewErrorStack()

	stack.Push(fmt.Errorf("first"))
	stack.Push(fmt.Errorf("second"))
	stack.Push(fmt.Errorf("third"))

	stack.SetFormat(ErrorStackFormatNumbered)
	errorString := stack.Error()
	expect.Equal("1: first\n2: second\n3: third", errorString)

	stack.SetFormat(ErrorStackFormatNewline)
	errorString = stack.Error()
	expect.Equal("first\nsecond\nthird", errorString)

	stack.SetFormat(ErrorStackFormatCSV)
	errorString = stack.Error()
	expect.Equal("first, second, third", errorString)
}
