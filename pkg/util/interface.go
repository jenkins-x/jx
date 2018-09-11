package util

import (
	"time"

	"github.com/cenkalti/backoff"
)

// Commander defines the interface for a Command
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/util Commander -o mocks/commander.go
type Commander interface {
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
