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
	"fmt"
	"github.com/trivago/tgo/tcontainer"
	"strings"
)

// ParserFlag is an enum type for flags used in parser transitions.
type ParserFlag int

// ParserStateID is used as an integer-based reference to a specific parser state.
// You can use any number (e.g. a hash) as a parser state representative.
type ParserStateID uint32

// ParsedFunc defines a function that a token has been matched
type ParsedFunc func([]byte, ParserStateID)

const (
	// ParserFlagContinue continues parsing at the position of the match.
	// By default the matched token will be skipped. This flag prevents the
	// default behavior. In addition to that the parser will add the parsed
	// token to the value of the next match.
	ParserFlagContinue = ParserFlag(1 << iota)

	// ParserFlagAppend causes the parser to keep the current value for the next
	// match. By default a value will be restarted after each match. This flag
	// prevents the default behavior.
	ParserFlagAppend = ParserFlag(1 << iota)

	// ParserFlagInclude includes the matched token in the read value.
	// By default the value does not contain the matched token.
	ParserFlagInclude = ParserFlag(1 << iota)

	// ParserFlagPush pushes the current state on the stack before making the
	// transition.
	ParserFlagPush = ParserFlag(1 << iota)

	// ParserFlagPop pops the stack and uses the returned state as the next
	// state. If no state is on the stack the regular next state is used.
	ParserFlagPop = ParserFlag(1 << iota)

	// ParserStateStop defines a state that causes the parsing to stop when
	// transitioned into.
	ParserStateStop = ParserStateID(0xFFFFFFFF)
)

// Transition defines a token based state change
type Transition struct {
	nextState ParserStateID
	flags     ParserFlag
	callback  ParsedFunc
}

// TransitionDirective contains a transition description that can be passed to
// the AddDirectives functions.
type TransitionDirective struct {
	State     string
	Token     string
	NextState string
	Flags     ParserFlag
	Callback  ParsedFunc
}

// TransitionParser defines the behavior of a parser by storing transitions from
// one state to another.
type TransitionParser struct {
	lookup []string
	tokens []*tcontainer.TrieNode
	stack  []ParserStateID
}

// NewTransition creates a new transition to a given state.
func NewTransition(nextState ParserStateID, flags ParserFlag, callback ParsedFunc) Transition {
	return Transition{
		nextState: nextState,
		flags:     flags,
		callback:  callback,
	}
}

// NewTransitionParser creates a new transition based parser
func NewTransitionParser() TransitionParser {
	return TransitionParser{
		lookup: []string{},
		tokens: []*tcontainer.TrieNode{},
		stack:  []ParserStateID{},
	}
}

// ParseTransitionDirective creates a transition directive from a string.
// This string must be of the following format:
//
// State:Token:NextState:Flag,Flag,...:Function
//
// Spaces will be stripped from all fields but Token. If a fields requires a
// colon it has to be escaped with a backslash. Other escape characters
// supported are \n, \r and \t.
// Flag can be a set of the
// following flags (input will be converted to lowercase):
//
//  * continue -> ParserFlagContinue
//  * append   -> ParserFlagAppend
//  * include  -> ParserFlagInclude
//  * push     -> ParserFlagPush
//  * pop      -> ParserFlagPop
//
// The name passed to the function token must be in the callbacks map. If it is
// not the data of the token will not be written. I.e. in empty string equals
// "do not write" here.
// An empty NextState will be translated to the "Stop" state as ususal.
func ParseTransitionDirective(config string, callbacks map[string]ParsedFunc) (TransitionDirective, error) {
	escape := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t", "\\:", "\b")
	restore := strings.NewReplacer("\b", ":")

	tokens := strings.Split(escape.Replace(config), ":")
	if len(tokens) != 5 {
		return TransitionDirective{}, fmt.Errorf("Parser directive requires 5 tokens.")
	}

	// Restore colons
	for i, token := range tokens {
		tokens[i] = restore.Replace(token)
	}

	flags := ParserFlag(0)
	flagStrings := strings.Split(tokens[3], ",")

	for _, flagName := range flagStrings {
		switch strings.ToLower(strings.TrimSpace(flagName)) {
		case "continue":
			flags |= ParserFlagContinue
		case "append":
			flags |= ParserFlagAppend
		case "include":
			flags |= ParserFlagInclude
		case "push":
			flags |= ParserFlagPush
		case "pop":
			flags |= ParserFlagPop
		}
	}

	callback, _ := callbacks[strings.TrimSpace(tokens[4])]
	dir := TransitionDirective{
		State:     strings.TrimSpace(tokens[0]),
		Token:     tokens[1],
		NextState: strings.TrimSpace(tokens[2]),
		Flags:     flags,
		Callback:  callback,
	}

	return dir, nil
}

// GetStateID creates a hash from the given state name.
// Empty state names will be translated to ParserStateStop.
func (parser *TransitionParser) GetStateID(stateName string) ParserStateID {
	if len(stateName) == 0 {
		return ParserStateStop
	}

	for id, name := range parser.lookup {
		if name == stateName {
			return ParserStateID(id) // ### return, found ###
		}
	}

	id := ParserStateID(len(parser.lookup))
	parser.lookup = append(parser.lookup, stateName)
	parser.tokens = append(parser.tokens, nil)
	return id
}

// GetStateName returns the name for the given state id or an empty string if
// the id could not be found.
func (parser *TransitionParser) GetStateName(id ParserStateID) string {
	if id < ParserStateID(len(parser.lookup)) {
		return parser.lookup[id]
	}
	return ""
}

// AddDirectives is a convenience function to add multiple transitions in as a
// batch.
func (parser *TransitionParser) AddDirectives(directives []TransitionDirective) {
	for _, dir := range directives {
		parser.Add(dir.State, dir.Token, dir.NextState, dir.Flags, dir.Callback)
	}
}

// Add adds a new transition to a given parser state.
func (parser *TransitionParser) Add(stateName string, token string, nextStateName string, flags ParserFlag, callback ParsedFunc) {
	nextStateID := parser.GetStateID(nextStateName)
	parser.AddTransition(stateName, NewTransition(nextStateID, flags, callback), token)
}

// Stop adds a stop transition to a given parser state.
func (parser *TransitionParser) Stop(stateName string, token string, flags ParserFlag, callback ParsedFunc) {
	parser.AddTransition(stateName, NewTransition(ParserStateStop, flags, callback), token)
}

// AddTransition adds a transition from a given state to the map
func (parser *TransitionParser) AddTransition(stateName string, newTrans Transition, token string) {
	stateID := parser.GetStateID(stateName)

	if state := parser.tokens[stateID]; state == nil {
		parser.tokens[stateID] = tcontainer.NewTrie([]byte(token), newTrans)
	} else {
		parser.tokens[stateID] = state.Add([]byte(token), newTrans)
	}
}

// Parse starts parsing at a given stateID.
// This function returns the remaining parts of data that did not match a
// transition as well as the last state the parser has been set to.
func (parser *TransitionParser) Parse(data []byte, state string) ([]byte, ParserStateID) {
	currentStateID := parser.GetStateID(state)

	if currentStateID == ParserStateStop {
		return nil, currentStateID // ### return, immediate stop ###
	}

	currentState := parser.tokens[currentStateID]
	dataLen := len(data)
	readStartIdx := 0
	continueIdx := 0

	for parseIdx := 0; parseIdx < dataLen; parseIdx++ {
		node := currentState.MatchStart(data[parseIdx:])
		if node == nil {
			continue // ### continue, no match ###
		}

		t := node.Payload.(Transition)
		if t.callback != nil {
			if t.flags&ParserFlagInclude != 0 {
				t.callback(data[readStartIdx:parseIdx+node.PathLen], currentStateID)
			} else {
				t.callback(data[readStartIdx:parseIdx], currentStateID)
			}
		}

		// Move the reader
		if t.flags&ParserFlagContinue == 0 {
			parseIdx += node.PathLen - 1
		} else {
			parseIdx--
		}
		continueIdx = parseIdx + 1

		if t.flags&ParserFlagAppend == 0 {
			readStartIdx = continueIdx
		}

		nextStateID := t.nextState

		// Pop before push to allow both at the same time
		if t.flags&ParserFlagPop != 0 {
			stackLen := len(parser.stack)
			if stackLen > 0 {
				nextStateID = parser.stack[stackLen-1]
				parser.stack = parser.stack[:stackLen-1]
			}
		}

		if t.flags&ParserFlagPush != 0 {
			parser.stack = append(parser.stack, currentStateID)
		}

		// Transition to next
		currentStateID = nextStateID
		if currentStateID == ParserStateStop {
			break // ### break, stop state ###
		}

		currentState = parser.tokens[currentStateID]
	}

	if readStartIdx == dataLen {
		return nil, currentStateID // ### return, everything parsed ###
	}

	return data[readStartIdx:], currentStateID
}
