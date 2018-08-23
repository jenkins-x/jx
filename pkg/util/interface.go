package util

import (
	"time"

	"github.com/cenkalti/backoff"
)

// CommandInterface defines the interface for a Command
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/util CommandInterface -o mocks/command_interface.go -p util_mock_test
type CommandInterface interface {
	DidError() bool
	DidFail() bool
	Error() error
	Run() (string, error)
	RunWithoutRetry() (string, error)
	SetName(string)
	SetDir(string)
	SetArgs([]string)
	SetTimeout(time.Duration)
	SetExponentialBackOff(*backoff.ExponentialBackOff)
}
