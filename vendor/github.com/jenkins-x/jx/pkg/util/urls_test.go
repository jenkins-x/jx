package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUrlJoin(t *testing.T) {
	assert.Equal(t, "http://foo.bar/whatnot/thingy", UrlJoin("http://foo.bar", "whatnot", "thingy"))
	assert.Equal(t, "http://foo.bar/whatnot/thingy/", UrlJoin("http://foo.bar/", "/whatnot/", "/thingy/"))
}
