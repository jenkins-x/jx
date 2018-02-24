package convh

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuessPrimitiveType(t *testing.T) {
	tests := []struct {
		str      string
		expected interface{}
	}{
		{
			"test",
			"test",
		},
		{
			"true",
			true,
		},
		{
			"3.4",
			float64(3.4),
		},
		{
			"13",
			int64(13),
		},
	}

	for _, test := range tests {
		assert.EqualValues(t, test.expected, GuessPrimitiveType(test.str))
	}
}

func TestConvertToString(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
		err      error
	}{
		{
			"test",
			"test",
			nil,
		},
		{
			true,
			"true",
			nil,
		},
		{
			float64(3.4),
			"3.4",
			nil,
		},
		{
			int(13),
			"13",
			nil,
		},
		{
			struct{}{},
			"",
			fmt.Errorf("value can't be converted to string"),
		},
	}

	for _, test := range tests {
		v, err := ConvertToString(test.value)

		assert.Equal(t, test.err, err)
		assert.EqualValues(t, test.expected, v)
	}
}
