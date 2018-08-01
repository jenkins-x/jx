package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDigitSuffix(t *testing.T) {
	testData := map[string]string{
		"nosuffix": "",
		"build1":   "1",
		"build123": "123",
	}

	for input, expected := range testData {
		actual := digitSuffix(input)
		assert.Equal(t, expected, actual, "digitSuffix for %s", input)
	}
}
