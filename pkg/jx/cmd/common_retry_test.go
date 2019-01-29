package cmd

import (
	"github.com/pkg/errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSuccessfulTry(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := (&CommonOptions{}).retryUntilFatalError(3, time.Millisecond * 50, func() (fatalError *FatalError, e error) {
		attempts++
		return nil, nil
	})

	assert.Nil(t, err)
	assert.Equal(t, 1, attempts)
}

func TestUnsuccessfulTry(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := (&CommonOptions{}).retryUntilFatalError(3, time.Millisecond * 50, func() (fatalError *FatalError, e error) {
		attempts++
		return nil, errors.New("invalid attempt")
	})

	assert.NotNil(t, err)
	assert.Equal(t, 3, attempts)
}

func TestSuccessfulAfterSecondAttempt(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := (&CommonOptions{}).retryUntilFatalError(3, time.Millisecond * 50, func() (fatalError *FatalError, e error) {
		attempts++
		if attempts == 2 {
			return nil, nil
		}
		return nil, errors.New("invalid attempt")
	})

	assert.Nil(t, err)
	assert.Equal(t, 2, attempts)
}


func TestFatal(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := (&CommonOptions{}).retryUntilFatalError(3, time.Millisecond * 50, func() (fatalError *FatalError, e error) {
		attempts++
		if attempts == 2 {
			return &FatalError{E: errors.New("fatal error")}, nil
		}
		return nil, errors.New("invalid attempt")
	})

	assert.NotNil(t, err)
	assert.Equal(t, 2, attempts)
}