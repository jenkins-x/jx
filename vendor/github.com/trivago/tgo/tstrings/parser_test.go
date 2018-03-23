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

package tstrings

import (
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
	"testing"
)

const (
	testStateSearchName  = ">name"
	testStateName        = "name"
	testStateSearchValue = ">value"
	testStateValue       = "value"
	testStateArray       = "array"
)

type parserTestState struct {
	currentName string
	parsed      map[string]interface{}
}

func (t *parserTestState) parsedName(data []byte, state ParserStateID) {
	t.currentName = string(data)
}

func (t *parserTestState) parsedValue(data []byte, state ParserStateID) {
	t.parsed[t.currentName] = string(data)
}

func (t *parserTestState) parsedArray(data []byte, state ParserStateID) {
	if value, exists := t.parsed[t.currentName]; !exists {
		t.parsed[t.currentName] = []string{string(data)}
	} else {
		array := value.([]string)
		t.parsed[t.currentName] = append(array, string(data))
	}
}

func TestTrie(t *testing.T) {
	expect := ttesting.NewExpect(t)

	root := tcontainer.NewTrie([]byte("abcd"), new(int))
	root = root.Add([]byte("abd"), new(int))
	root = root.Add([]byte("cde"), new(int))

	node := root.Match([]byte("abcd"))
	if expect.NotNil(node) {
		expect.NotNil(node.Payload)
		expect.Equal(4, node.PathLen)
	}

	node = root.Match([]byte("ab"))
	expect.Nil(node)

	node = root.MatchStart([]byte("abcdef"))
	if expect.NotNil(node) {
		expect.NotNil(node.Payload)
		expect.Equal(4, node.PathLen)
	}

	node = root.MatchStart([]byte("bcde"))
	expect.Nil(node)

	root2 := tcontainer.NewTrie([]byte("a"), new(int))
	root2 = root.Add([]byte("b"), new(int))
	root2 = root.Add([]byte("c"), new(int))

	node = root2.Match([]byte("c"))
	if expect.NotNil(node) {
		expect.NotNil(node.Payload)
		expect.Equal(1, node.PathLen)
	}
}

func TestParser(t *testing.T) {
	state := parserTestState{parsed: make(map[string]interface{})}
	expect := ttesting.NewExpect(t)

	dir := []TransitionDirective{
		{testStateSearchName, `"`, testStateName, 0, nil},
		{testStateSearchName, `}`, "", 0, nil},
		{testStateName, `"`, testStateSearchValue, 0, state.parsedName},
		{testStateSearchValue, `:`, testStateValue, 0, nil},
		{testStateValue, `[`, testStateArray, 0, nil},
		{testStateValue, `,`, testStateSearchName, 0, state.parsedValue},
		{testStateValue, `}`, "", 0, state.parsedValue},
		{testStateArray, `,`, testStateArray, 0, state.parsedArray},
		{testStateArray, `],`, testStateSearchName, 0, state.parsedArray},
	}

	parser := NewTransitionParser()
	parser.AddDirectives(dir)

	dataTest := `{"test":123,"array":[a,b,c],"end":456}`
	parser.Parse([]byte(dataTest), testStateSearchName)

	expect.MapSet(state.parsed, "test")
	expect.MapSet(state.parsed, "array")
	expect.MapSet(state.parsed, "end")

	expect.MapEqual(state.parsed, "test", "123")
	expect.MapEqual(state.parsed, "array", []string{"a", "b", "c"})
	expect.MapEqual(state.parsed, "end", "456")
}

func TestDirectiveParser(t *testing.T) {
	expect := ttesting.NewExpect(t)
	callbacks := make(map[string]ParsedFunc)

	callbacks["write"] = func(data []byte, state ParserStateID) {
	}

	directive, err := ParseTransitionDirective("start:>:::", callbacks)
	if expect.Nil(err) {
		expect.Equal("start", directive.State)
		expect.Equal(">", directive.Token)
		expect.Equal("", directive.NextState)
		expect.Equal(0, int(directive.Flags))
		expect.Nil(directive.Callback)
	}

	directive, err = ParseTransitionDirective(" start : \\:: next : continue : write", callbacks)
	if expect.Nil(err) {
		expect.Equal("start", directive.State)
		expect.Equal(" :", directive.Token)
		expect.Equal("next", directive.NextState)
		expect.Equal(int(ParserFlagContinue), int(directive.Flags))
		expect.NotNil(directive.Callback)
	}
}
