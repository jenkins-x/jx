package counter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounter(t *testing.T) {
	c := Counter{}
	assert.Equal(t, c.Get(), "0")

	c.Increment()
	assert.Equal(t, c.Get(), "1")
}
