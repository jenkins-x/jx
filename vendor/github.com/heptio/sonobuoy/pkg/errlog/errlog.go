/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errlog

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// DebugOutput controls whether to output the trace of every error
var DebugOutput = false

// LogError logs an error, optionally with a tracelog
func LogError(err error) {
	if DebugOutput {
		// Print the error message with the stack trace (%+v) in the "trace" field
		logrus.WithField("trace", fmt.Sprintf("%+v", err)).Error(err)
	} else {
		logrus.Error(err.Error())
	}
}
