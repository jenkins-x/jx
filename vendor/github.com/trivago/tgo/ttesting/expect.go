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

package ttesting

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var expectBasePath string

func init() {
	pkgPath := reflect.TypeOf(Expect{}).PkgPath()
	pathEndIdx := strings.LastIndex(pkgPath, "/")
	expectBasePath = pkgPath[:pathEndIdx]
}

// Expect is a helper construct for unittesting
type Expect struct {
	scope  string
	group  string
	t      *testing.T
	silent bool
}

// NewExpect creates an Expect helper struct with scope set to the name of the
// calling function.
func NewExpect(t *testing.T) Expect {
	pc, _, _, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc)
	funcName := caller.Name()

	return Expect{
		scope:  funcName[strings.LastIndex(funcName, ".")+1:],
		t:      t,
		silent: false,
	}
}

// NewSilentExpect works like NewExpect but surpresses fails and error messages.
// This is usefull to do inverse testing, i.e. checking if a check fails.
func NewSilentExpect(t *testing.T) Expect {
	pc, _, _, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc)
	funcName := caller.Name()

	return Expect{
		scope:  funcName[strings.LastIndex(funcName, ".")+1:],
		t:      t,
		silent: true,
	}
}

// StartGroup starts a new test groups. Errors will be prefixed with the name of
// this group until EndGroup is called.
func (e *Expect) StartGroup(name string) {
	e.group = name
}

// EndGroup closes a test group. See StartGroup.
func (e *Expect) EndGroup() {
	e.group = ""
}

func (e Expect) error(message string) {
	if e.silent {
		return
	}
	_, file, line, _ := runtime.Caller(2)
	if basePathIdx := strings.Index(file, expectBasePath); basePathIdx > -1 {
		file = file[basePathIdx+len(expectBasePath):]
	}

	scope := e.scope
	if e.group != "" {
		scope += "/" + e.group
	}

	fmt.Printf("\t%s:%d: %s -> %s\n", file, line, scope, message)
	e.t.Fail()
}

func (e Expect) errorf(format string, args ...interface{}) {
	if e.silent {
		return
	}
	_, file, line, _ := runtime.Caller(2)
	if basePathIdx := strings.Index(file, expectBasePath); basePathIdx > -1 {
		file = file[basePathIdx+len(expectBasePath):]
	}

	fmt.Printf(fmt.Sprintf("\t%s:%d: %s -> %s\n", file, line, e.scope, format), args...)
	e.t.Fail()
}

// NotExecuted always reports an error when called
func (e Expect) NotExecuted() {
	e.error("This was expected not to be called")
}

// True tests if the given value is true
func (e Expect) True(val bool) bool {
	if !val {
		e.error("Expected true")
		return false
	}
	return true
}

// False tests if the given value is false
func (e Expect) False(val bool) bool {
	if val {
		e.error("Expected false")
		return false
	}
	return true
}

// Nil tests if the given value is nil
func (e Expect) Nil(val interface{}) bool {
	rval := reflect.ValueOf(val)
	switch rval.Kind() {
	case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if !rval.IsNil() {
			e.error("Expected nil")
			return false
		}

	default:
		if val != nil {
			e.error("Expected nil")
			return false
		}
	}

	return true
}

// NotNil tests if the given value is not nil
func (e Expect) NotNil(val interface{}) bool {
	if val == nil {
		e.error("Expected not nil")
		return false
	}

	rval := reflect.ValueOf(val)
	switch rval.Kind() {
	case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if rval.IsNil() {
			e.error("Expected not nil")
			return false
		}
	}
	return true
}

// NoError tests if the given error is nil. If it is not the error will be
// logged.
func (e Expect) NoError(err error) bool {
	if err != nil {
		e.error("Expected no error, got " + err.Error())
		return false
	}
	return true
}

// NoPanic tests if the given function panics. If it does, the panic will be
// logged.
func (e Expect) NoPanic(callback func()) {
	defer func() {
		if r := recover(); r != nil {
			e.errorf("Expected no panic, got %#v", r)
		}
	}()

	callback()
}

// Contains returns true if val1 contains val2
func (e Expect) Contains(val1, val2 interface{}) bool {
	val1Type := reflect.TypeOf(val1)
	switch val1Type.Kind() {
	case reflect.String:
		str1 := val1.(string)
		str2, isString := val2.(string)
		if !isString {
			e.errorf("Second argument to contains is no string")
			return false
		}
		if !strings.Contains(str1, str2) {
			e.errorf("Expected %s to contain %s", val1, val2)
			return false
		}
		return true

	case reflect.Array, reflect.Slice:
		slice := reflect.ValueOf(val1)
		for i := 0; i < slice.Len(); i++ {
			item := slice.Index(i).Interface()
			if reflect.DeepEqual(item, val2) {
				return true
			}
		}
		e.errorf("Expected %#v to contain %#v", val1, val2)
		return false

	default:
		e.errorf("Contains expects a string or slice")
		return false
	}
}

// ContainsN returns true if val1 contains val2 count times
func (e Expect) ContainsN(val1, val2 interface{}, count int) bool {
	val1Type := reflect.TypeOf(val1)
	switch val1Type.Kind() {
	case reflect.String:
		str1 := val1.(string)
		str2, isString := val2.(string)
		if !isString {
			e.errorf("Second argument to contains is no string")
			return false
		}
		if times := strings.Count(str1, str2); times != count {
			e.errorf("Expected %s to contain %s %d times. Found %d occurences", val1, val2, count, times)
			return false
		}
		return true

	case reflect.Array, reflect.Slice:
		slice := reflect.ValueOf(val1)
		times := 0
		for i := 0; i < slice.Len(); i++ {
			item := slice.Index(i).Interface()
			if reflect.DeepEqual(item, val2) {
				times++
			}
		}
		if times != count {
			e.errorf("Expected %#v to contain %#v %d times. Found %d occurences", val1, val2, count, times)
			return false
		}
		return true

	default:
		e.errorf("Contains expects a string or slice")
		return false
	}
}

// Equal does a deep equality check on both values and returns true if that test
// yielded true (val1 == val2)
func (e Expect) Equal(val1, val2 interface{}) bool {
	if !reflect.DeepEqual(val1, val2) {
		e.errorf("Expected %T(%v), got %T(%v)", val1, val1, val2, val2)
		return false
	}
	return true
}

// Neq does a deep equality check on both values and returns true if that test
// yielded false (val1 != val2)
func (e Expect) Neq(val1, val2 interface{}) bool {
	if reflect.DeepEqual(val1, val2) {
		e.errorf("Expected not %T(%v)", val1, val1)
		return false
	}
	return true
}

// Greater does a numeric greater than check on both values and returns true if
// that test yielded true (val1 > val2)
func (e Expect) Greater(val1, val2 interface{}) bool {
	if reflect.TypeOf(val1) != reflect.TypeOf(val2) {
		e.errorf("Expect.Greater requires both values to be of the same type. Got %T and %T.", val1, val2)
		return false
	}

	switch val1.(type) {
	case int:
		if val1.(int) <= val2.(int) {
			e.errorf("%d <= %d.", val1.(int), val2.(int))
			return false
		}
	case int8:
		if val1.(int8) <= val2.(int8) {
			e.errorf("%d <= %d.", val1.(int8), val2.(int8))
			return false
		}
	case int16:
		if val1.(int16) <= val2.(int16) {
			e.errorf("%d <= %d.", val1.(int16), val2.(int16))
			return false
		}
	case int32:
		if val1.(int32) <= val2.(int32) {
			e.errorf("%d <= %d.", val1.(int32), val2.(int32))
			return false
		}
	case int64:
		if val1.(int64) <= val2.(int64) {
			e.errorf("%d <= %d.", val1.(int64), val2.(int64))
			return false
		}
	case uint:
		if val1.(uint) <= val2.(uint) {
			e.errorf("%d <= %d.", val1.(uint), val2.(uint))
			return false
		}
	case uint8:
		if val1.(uint8) <= val2.(uint8) {
			e.errorf("%d <= %d.", val1.(uint8), val2.(uint8))
			return false
		}
	case uint16:
		if val1.(uint16) <= val2.(uint16) {
			e.errorf("%d <= %d.", val1.(uint16), val2.(uint16))
			return false
		}
	case uint32:
		if val1.(uint32) <= val2.(uint32) {
			e.errorf("%d <= %d.", val1.(uint32), val2.(uint32))
			return false
		}
	case uint64:
		if val1.(uint64) <= val2.(uint64) {
			e.errorf("%d <= %d.", val1.(uint64), val2.(uint64))
			return false
		}
	case float32:
		if val1.(float32) <= val2.(float32) {
			e.errorf("%f <= %f.", val1.(float32), val2.(float32))
			return false
		}
	case float64:
		if val1.(float64) <= val2.(float64) {
			e.errorf("%f <= %f.", val1.(float64), val2.(float64))
			return false
		}
	default:
		e.errorf("Cannot test %T for \"greater than\".", val1)
		return false
	}

	return true
}

// Geq does a numeric greater or equal check on both values and returns true if
// that test yielded true (val1 >= val2)
func (e Expect) Geq(val1, val2 interface{}) bool {
	if reflect.TypeOf(val1) != reflect.TypeOf(val2) {
		e.errorf("Expect.Geq requires both values to be of the same type. Got %T and %T.", val1, val2)
		return false
	}

	switch val1.(type) {
	case int:
		if val1.(int) < val2.(int) {
			e.errorf("%d < %d.", val1.(int), val2.(int))
			return false
		}
	case int8:
		if val1.(int8) < val2.(int8) {
			e.errorf("%d < %d.", val1.(int8), val2.(int8))
			return false
		}
	case int16:
		if val1.(int16) < val2.(int16) {
			e.errorf("%d < %d.", val1.(int16), val2.(int16))
			return false
		}
	case int32:
		if val1.(int32) < val2.(int32) {
			e.errorf("%d < %d.", val1.(int32), val2.(int32))
			return false
		}
	case int64:
		if val1.(int64) < val2.(int64) {
			e.errorf("%d < %d.", val1.(int64), val2.(int64))
			return false
		}
	case uint:
		if val1.(uint) < val2.(uint) {
			e.errorf("%d < %d.", val1.(uint), val2.(uint))
			return false
		}
	case uint8:
		if val1.(uint8) < val2.(uint8) {
			e.errorf("%d < %d.", val1.(uint8), val2.(uint8))
			return false
		}
	case uint16:
		if val1.(uint16) < val2.(uint16) {
			e.errorf("%d < %d.", val1.(uint16), val2.(uint16))
			return false
		}
	case uint32:
		if val1.(uint32) < val2.(uint32) {
			e.errorf("%d < %d.", val1.(uint32), val2.(uint32))
			return false
		}
	case uint64:
		if val1.(uint64) < val2.(uint64) {
			e.errorf("%d < %d.", val1.(uint64), val2.(uint64))
			return false
		}
	case float32:
		if val1.(float32) < val2.(float32) {
			e.errorf("%f < %f.", val1.(float32), val2.(float32))
			return false
		}
	case float64:
		if val1.(float64) < val2.(float64) {
			e.errorf("%f < %f.", val1.(float64), val2.(float64))
			return false
		}
	default:
		e.errorf("Cannot test %T for \"greater or equal\".", val1)
		return false
	}

	return true
}

// Less does a numeric less than check on both values and returns true if
// that test yielded true (val1 < val2)
func (e Expect) Less(val1, val2 interface{}) bool {
	if reflect.TypeOf(val1) != reflect.TypeOf(val2) {
		e.errorf("Expect.Less requires both values to be of the same type. Got %T and %T.", val1, val2)
		return false
	}

	switch val1.(type) {
	case int:
		if val1.(int) >= val2.(int) {
			e.errorf("%d >= %d.", val1.(int), val2.(int))
			return false
		}
	case int8:
		if val1.(int8) >= val2.(int8) {
			e.errorf("%d >= %d.", val1.(int8), val2.(int8))
			return false
		}
	case int16:
		if val1.(int16) >= val2.(int16) {
			e.errorf("%d >= %d.", val1.(int16), val2.(int16))
			return false
		}
	case int32:
		if val1.(int32) >= val2.(int32) {
			e.errorf("%d >= %d.", val1.(int32), val2.(int32))
			return false
		}
	case int64:
		if val1.(int64) >= val2.(int64) {
			e.errorf("%d >= %d.", val1.(int64), val2.(int64))
			return false
		}
	case uint:
		if val1.(uint) >= val2.(uint) {
			e.errorf("%d >= %d.", val1.(uint), val2.(uint))
			return false
		}
	case uint8:
		if val1.(uint8) >= val2.(uint8) {
			e.errorf("%d >= %d.", val1.(uint8), val2.(uint8))
			return false
		}
	case uint16:
		if val1.(uint16) >= val2.(uint16) {
			e.errorf("%d >= %d.", val1.(uint16), val2.(uint16))
			return false
		}
	case uint32:
		if val1.(uint32) >= val2.(uint32) {
			e.errorf("%d >= %d.", val1.(uint32), val2.(uint32))
			return false
		}
	case uint64:
		if val1.(uint64) >= val2.(uint64) {
			e.errorf("%d >= %d.", val1.(uint64), val2.(uint64))
			return false
		}
	case float32:
		if val1.(float32) >= val2.(float32) {
			e.errorf("%f >= %f.", val1.(float32), val2.(float32))
			return false
		}
	case float64:
		if val1.(float64) >= val2.(float64) {
			e.errorf("%f >= %f.", val1.(float64), val2.(float64))
			return false
		}
	default:
		e.errorf("Cannot test %T for \"less than\".", val1)
		return false
	}

	return true
}

// Leq does a numeric less or euqal check on both values and returns true if
// that test yielded true (val1 <= val2)
func (e Expect) Leq(val1, val2 interface{}) bool {
	if reflect.TypeOf(val1) != reflect.TypeOf(val2) {
		e.errorf("Expect.Leq requires both values to be of the same type. Got %T and %T.", val1, val2)
		return false
	}

	switch val1.(type) {
	case int:
		if val1.(int) > val2.(int) {
			e.errorf("%d > %d.", val1.(int), val2.(int))
			return false
		}
	case int8:
		if val1.(int8) > val2.(int8) {
			e.errorf("%d > %d.", val1.(int8), val2.(int8))
			return false
		}
	case int16:
		if val1.(int16) > val2.(int16) {
			e.errorf("%d > %d.", val1.(int16), val2.(int16))
			return false
		}
	case int32:
		if val1.(int32) > val2.(int32) {
			e.errorf("%d > %d.", val1.(int32), val2.(int32))
			return false
		}
	case int64:
		if val1.(int64) > val2.(int64) {
			e.errorf("%d > %d.", val1.(int64), val2.(int64))
			return false
		}
	case uint:
		if val1.(uint) > val2.(uint) {
			e.errorf("%d > %d.", val1.(uint), val2.(uint))
			return false
		}
	case uint8:
		if val1.(uint8) > val2.(uint8) {
			e.errorf("%d > %d.", val1.(uint8), val2.(uint8))
			return false
		}
	case uint16:
		if val1.(uint16) > val2.(uint16) {
			e.errorf("%d > %d.", val1.(uint16), val2.(uint16))
			return false
		}
	case uint32:
		if val1.(uint32) > val2.(uint32) {
			e.errorf("%d > %d.", val1.(uint32), val2.(uint32))
			return false
		}
	case uint64:
		if val1.(uint64) > val2.(uint64) {
			e.errorf("%d > %d.", val1.(uint64), val2.(uint64))
			return false
		}
	case float32:
		if val1.(float32) > val2.(float32) {
			e.errorf("%f > %f.", val1.(float32), val2.(float32))
			return false
		}
	case float64:
		if val1.(float64) > val2.(float64) {
			e.errorf("%f > %f.", val1.(float64), val2.(float64))
			return false
		}
	default:
		e.errorf("Cannot test %T for \"less or euqal\".", val1)
		return false
	}

	return true
}

// OfType returns true if both values are of the same type
func (e Expect) OfType(t interface{}, val interface{}) bool {
	if reflect.TypeOf(t) != reflect.TypeOf(val) {
		e.errorf("Expected type \"%t\" but found \"%t\"", t, val)
		return false
	}
	return true
}

// MapSet returns true if the key is set in the given array
func (e Expect) MapSet(data interface{}, key interface{}) bool {
	mapVal := reflect.ValueOf(data)
	keyVal := reflect.ValueOf(key)
	value := mapVal.MapIndex(keyVal)
	if !value.IsValid() {
		e.errorf("Expected key \"%v\" to be set", key)
		return false
	}
	return true
}

// MapNotSet returns true if the key is not set in the given array
func (e Expect) MapNotSet(data interface{}, key interface{}) bool {
	mapVal := reflect.ValueOf(data)
	keyVal := reflect.ValueOf(key)
	value := mapVal.MapIndex(keyVal)
	if value.IsValid() {
		e.errorf("Expected key \"%v\" not to be set", key)
		return false
	}
	return true
}

// MapEqual returns true if MapSet equals true for the given key and Equal
// returns true for the returned and given value.
func (e Expect) MapEqual(mapData interface{}, key interface{}, value interface{}) bool {
	mapVal := reflect.ValueOf(mapData)
	keyVal := reflect.ValueOf(key)
	valVal := mapVal.MapIndex(keyVal)
	if !valVal.IsValid() {
		e.errorf("Expected key \"%v\" to be set", key)
		return false
	}

	return e.Equal(value, valVal.Interface())
}

// MapNeq returns true if MapSet equals true for the given key and Neq returns
// true for the returned and given value.
func (e Expect) MapNeq(data interface{}, key interface{}, value interface{}) bool {
	mapVal := reflect.ValueOf(data)
	keyVal := reflect.ValueOf(key)
	valVal := mapVal.MapIndex(keyVal)
	if !valVal.IsValid() {
		e.errorf("Expected key \"%v\" to be set", key)
		return false
	}

	return e.Neq(value, valVal.Interface())
}

// MapLess returns true if MapSet equals true for the given key and Less returns
// true for the returned and given value.
func (e Expect) MapLess(data interface{}, key interface{}, value interface{}) bool {
	mapVal := reflect.ValueOf(data)
	keyVal := reflect.ValueOf(key)
	valVal := mapVal.MapIndex(keyVal)
	if !valVal.IsValid() {
		e.errorf("Expected key \"%v\" to be set", key)
		return false
	}

	return e.Less(value, valVal.Interface())
}

// MapGreater returns true if MapSet equals true for the given key and Greater
// returns true for the returned and given value.
func (e Expect) MapGreater(data interface{}, key interface{}, value interface{}) bool {
	mapVal := reflect.ValueOf(data)
	keyVal := reflect.ValueOf(key)
	valVal := mapVal.MapIndex(keyVal)
	if !valVal.IsValid() {
		e.errorf("Expected key \"%v\" to be set", key)
		return false
	}

	return e.Greater(value, valVal.Interface())
}

// MapLeq returns true if MapSet equals true for the given key and Leq returns
// true for the returned and given value.
func (e Expect) MapLeq(data interface{}, key interface{}, value interface{}) bool {
	mapVal := reflect.ValueOf(data)
	keyVal := reflect.ValueOf(key)
	valVal := mapVal.MapIndex(keyVal)
	if !valVal.IsValid() {
		e.errorf("Expected key \"%v\" to be set", key)
		return false
	}

	return e.Leq(value, valVal.Interface())
}

// MapGeq returns true if MapSet equals true for the given key and Geq returns
// true for the returned and given value.
func (e Expect) MapGeq(data interface{}, key interface{}, value interface{}) bool {
	mapVal := reflect.ValueOf(data)
	keyVal := reflect.ValueOf(key)
	valVal := mapVal.MapIndex(keyVal)
	if !valVal.IsValid() {
		e.errorf("Expected key \"%v\" to be set", key)
		return false
	}

	return e.Geq(value, valVal.Interface())
}

// NonBlocking waits for the given timeout for the routine to return. If
// timed out it is an error.
func (e Expect) NonBlocking(t time.Duration, routine func()) bool {
	cmd := make(chan struct{})

	started := new(sync.WaitGroup)
	started.Add(1)

	go func() {
		started.Done()
		routine()
		close(cmd)
	}()

	started.Wait() // wait for go routine to get scheduled

	select {
	case <-cmd:
		return true

	case <-time.After(t):
		e.errorf("Evaluation timed out.")
		return false
	}
}
