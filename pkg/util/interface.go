package util

import (
	"time"

	"github.com/cenkalti/backoff"
)

// Commander defines the interface for a Command
//go:generate pegomock generate github.com/jenkins-x/jx/v2/pkg/util Commander -o mocks/commander.go
type Commander interface {
	DidError() bool
	DidFail() bool
	Error() error
	Run() (string, error)
	RunWithoutRetry() (string, error)
	SetName(string)
	CurrentName() string
	SetDir(string)
	CurrentDir() string
	SetArgs([]string)
	CurrentArgs() []string
	SetTimeout(time.Duration)
	SetExponentialBackOff(*backoff.ExponentialBackOff)
	SetEnv(map[string]string)
	CurrentEnv() map[string]string
	SetEnvVariable(string, string)
}
