package cmd_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
)

func TestDigitSuffix(t *testing.T) {
	testData := map[string]string{
		"nosuffix": "",
		"build1":   "1",
		"build123": "123",
	}

	for input, expected := range testData {
		actual := cmd.DigitSuffix(input)
		assert.Equal(t, expected, actual, "digitSuffix for %s", input)
	}
}
