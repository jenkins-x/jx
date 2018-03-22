package tsync

import (
	"reflect"
)

// Fanout receives from the given in channel and forwards the data to the first
// non-blocking out channel. Fanout returns when in has been closed.
func Fanout(in interface{}, out ...interface{}) {
	cases := newSendCases(out)
	inChan := reflect.ValueOf(in)

	for {
		data, more := inChan.Recv()
		if !more {
			return
		}
		for i := range cases {
			cases[i].Send = data
		}
		reflect.Select(cases)
	}
}

// Funnel receives from the first non-blocking in channel and forwards it to the
// given out channel. Funnel returns when all in channels have been closed.
func Funnel(out interface{}, in ...interface{}) {
	cases := newRecvCases(in)
	outChan := reflect.ValueOf(out)

	for len(cases) > 0 {
		idx, val, ok := reflect.Select(cases)
		if !ok {
			cases = removeCase(idx, cases)
			continue
		}
		outChan.Send(val)
	}
}

// Turnout multiplexes data between the list of in and out channels. The data of
// the first non-blocking in channel will be forwarded to the first non-blocking
// out channel. Turnout returns when all in channels have been closed.
func Turnout(in []interface{}, out []interface{}) {
	inCases := newRecvCases(in)
	outCases := newSendCases(out)

	for len(inCases) > 0 {
		idx, val, ok := reflect.Select(inCases)
		if !ok {
			inCases = removeCase(idx, inCases)
			continue
		}

		for i := range outCases {
			outCases[i].Send = val
		}
		reflect.Select(outCases)
	}
}

func newSendCases(send []interface{}) []reflect.SelectCase {
	cases := make([]reflect.SelectCase, 0, len(send))
	for _, c := range send {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(c),
		})
	}
	return cases
}

func newRecvCases(recv []interface{}) []reflect.SelectCase {
	cases := make([]reflect.SelectCase, 0, len(recv))
	for _, c := range recv {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(c),
		})
	}
	return cases
}

func removeCase(i int, c []reflect.SelectCase) []reflect.SelectCase {
	switch {
	case len(c) == 0:
		return []reflect.SelectCase{}

	case i == 0:
		return c[1:]

	case i == len(c)-1:
		return c[:i]

	default:
		return append(c[:i], c[i+1:]...)
	}
}
