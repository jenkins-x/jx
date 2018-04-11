package gits

import (
	"testing"
	"time"

	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
)

func TestDateFormatAndParse(t *testing.T) {
	now := time.Now()

	expected := FormatDate(now)

	tests.Debugf("Formatted %s as %s\n", now.String(), expected)
	parsedTime, err := ParseDate(expected)
	assert.Nil(t, err)

	actual := FormatDate(parsedTime)

	assert.Equal(t, expected, actual, "Formatted dates for %s", now.String())
}
