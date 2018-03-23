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

package tlog

import (
	"github.com/trivago/tgo/tfmt"
	"io"
	"log"
	"os"
)

// Verbosity defines an enumeration for log verbosity
type Verbosity byte

const (
	// VerbosityError shows only error messages
	VerbosityError = Verbosity(iota)
	// VerbosityWarning shows error and warning messages
	VerbosityWarning = Verbosity(iota)
	// VerbosityNote shows error, warning and note messages
	VerbosityNote = Verbosity(iota)
	// VerbosityDebug shows all messages
	VerbosityDebug = Verbosity(iota)
)

var (
	// Error is a predefined log channel for errors. This log is backed by consumer.Log
	Error = log.New(logDisabled, "", log.Lshortfile)

	// Warning is a predefined log channel for warnings. This log is backed by consumer.Log
	Warning = log.New(logDisabled, "", 0)

	// Note is a predefined log channel for notes. This log is backed by consumer.Log
	Note = log.New(logDisabled, "", 0)

	// Debug is a predefined log channel for debug messages. This log is backed by consumer.Log
	Debug = log.New(logDisabled, "", 0)
)

var (
	logEnabled  = logReferrer{os.Stderr}
	logDisabled = logNull{}
)

// LogScope allows to wrap the standard Error, Warning, Note and Debug loggers
// into a scope, i.e. all messages written to this logger are prefixed.
type LogScope struct {
	Error   *log.Logger
	Warning *log.Logger
	Note    *log.Logger
	Debug   *log.Logger
	name    string
}

// NewLogScope creates a new LogScope with the given prefix string.
func NewLogScope(name string) LogScope {
	scopeMarker := tfmt.Colorizef(tfmt.DarkGray, tfmt.NoBackground, "[%s] ", name)

	return LogScope{
		name:    name,
		Error:   log.New(logLogger{Error}, scopeMarker, 0),
		Warning: log.New(logLogger{Warning}, scopeMarker, 0),
		Note:    log.New(logLogger{Note}, scopeMarker, 0),
		Debug:   log.New(logLogger{Debug}, scopeMarker, 0),
	}
}

// NewSubScope creates a log scope inside an existing log scope.
func (scope *LogScope) NewSubScope(name string) LogScope {
	scopeMarker := tfmt.Colorizef(tfmt.DarkGray, tfmt.NoBackground, "[%s.%s] ", scope.name, name)

	return LogScope{
		name:    name,
		Error:   log.New(logLogger{Error}, scopeMarker, 0),
		Warning: log.New(logLogger{Warning}, scopeMarker, 0),
		Note:    log.New(logLogger{Note}, scopeMarker, 0),
		Debug:   log.New(logLogger{Debug}, scopeMarker, 0),
	}
}

func init() {
	log.SetFlags(0)
	log.SetOutput(logEnabled)
	SetVerbosity(VerbosityError)
}

// SetVerbosity defines the type of messages to be processed.
// High level verobosities contain lower levels, i.e. log level warning will
// contain error messages, too.
func SetVerbosity(loglevel Verbosity) {
	Error = log.New(logDisabled, "", 0)
	Warning = log.New(logDisabled, "", 0)
	Note = log.New(logDisabled, "", 0)
	Debug = log.New(logDisabled, "", 0)

	switch loglevel {
	default:
		fallthrough

	case VerbosityDebug:
		Debug = log.New(&logEnabled, tfmt.Colorize(tfmt.Cyan, tfmt.NoBackground, "Debug: "), 0)
		fallthrough

	case VerbosityNote:
		Note = log.New(&logEnabled, "", 0)
		fallthrough

	case VerbosityWarning:
		Warning = log.New(&logEnabled, tfmt.Colorize(tfmt.Yellow, tfmt.NoBackground, "Warning: "), 0)
		fallthrough

	case VerbosityError:
		Error = log.New(&logEnabled, tfmt.Colorize(tfmt.Red, tfmt.NoBackground, "ERROR: "), log.Lshortfile)
	}
}

// SetCacheWriter will force all logs to be cached until another writer is set
func SetCacheWriter() {
	if _, isCache := logEnabled.writer.(*logCache); !isCache {
		logEnabled.writer = new(logCache)
	}
}

// SetWriter forces (enabled) logs to be written to the given writer.
func SetWriter(writer io.Writer) {
	oldWriter := logEnabled.writer
	logEnabled.writer = writer
	if cache, isCache := oldWriter.(*logCache); isCache {
		cache.Flush(logEnabled)
	}
}
