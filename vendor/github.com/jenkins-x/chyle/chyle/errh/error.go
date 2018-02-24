package errh

import (
	"fmt"
)

type errWrapper struct {
	msg string
	err error
}

func (e errWrapper) Error() string {
	return fmt.Sprintf("%s : %s", e.msg, e.err)
}

// AddCustomMessageToError appends a string message to an error
// by creating a brand new error
func AddCustomMessageToError(msg string, err error) error {
	if err == nil {
		return nil
	}

	return errWrapper{msg, err}
}
