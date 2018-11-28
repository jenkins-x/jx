package util

import (
	"fmt"
	"gopkg.in/AlecAivazis/survey.v1"
	"strings"
)

//NoWhiteSpaceValidator is an input validator for the survey package that disallows any whitespace in the input
func NoWhiteSpaceValidator() survey.Validator {
	// return a validator that ensures the given string does not contain any whitespace
	return func(val interface{}) error {
		if str, ok := val.(string); ok {
			if strings.ContainsAny(str, " ") {
				// yell loudly
				return fmt.Errorf("supplied value \"%v\" must not contain any whitespace", str)
			}
		}
		// the input is fine
		return nil
	}
}

