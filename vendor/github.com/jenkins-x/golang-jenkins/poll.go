package gojenkins

import (
	"fmt"
	"time"
)

// ConditionFunc returns true if the condition is satisfied, or an error
// if the loop should be aborted.
type ConditionFunc func() (done bool, err error)

// NewConditionFunc combines the functions into a single function that can be used when polling
func NewConditionFunc(functions ...ConditionFunc) ConditionFunc {
	return func() (bool, error) {
		for _, fn := range functions {
			done, err := fn()
			if done || err != nil {
				return done, err
			}
		}
		return false, nil
	}
}

// Poll polls the given function until it returns true to indicate its complete or an error
func Poll(pollPeriod time.Duration, timeout time.Duration, timeoutFailureMessage string, fn ConditionFunc) error {
	timeoutAt := time.Now().Add(timeout)
	useTimeout := timeout.Nanoseconds() > 0
	for {
		ok, err := fn()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		if useTimeout && time.Now().After(timeoutAt) {
			return fmt.Errorf("Timed out waiting for %s waited for %s", timeoutFailureMessage, timeout.String())
		}
		time.Sleep(pollPeriod)
	}
}
