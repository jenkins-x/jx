// Package counter provides a way to create a singleton to increment and store a counter
package counter

import (
	"strconv"
)

// Counter provides a way to generate and store a incremented id
type Counter struct {
	counter int
}

// Get returns current counter value
func (c *Counter) Get() string {
	return strconv.Itoa(c.counter)
}

// Increment adds one to actual counter value
func (c *Counter) Increment() {
	c.counter++
}
