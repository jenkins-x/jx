package util

import (
	"github.com/cenkalti/backoff"
	"time"
)

// Retry retries with exponential backoff the given function
func Retry(maxElapsedTime time.Duration, f func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = maxElapsedTime
	bo.Reset()
	return backoff.Retry(f, bo)

}
