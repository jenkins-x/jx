package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContentTypeForFile(t *testing.T) {
	t.Parallel()

	testData := map[string]string{
		"foo.log": "text/plain; charset=utf-8",
		"foo.txt": "text/plain; charset=utf-8",
		"foo.xml": "application/xml",
		"foo.json": "application/json",
	}

	for fileName, contentType := range testData {
		actual := ContentTypeForFileName(fileName)
		assert.Equal(t, contentType, actual, "content type for file %s", fileName)
	}
}

