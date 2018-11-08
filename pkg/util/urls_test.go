package util_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestUrlJoin(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "http://foo.bar/whatnot/thingy", util.UrlJoin("http://foo.bar", "whatnot", "thingy"))
	assert.Equal(t, "http://foo.bar/whatnot/thingy/", util.UrlJoin("http://foo.bar/", "/whatnot/", "/thingy/"))
}

func TestUrlHostNameWithoutPort(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"hostname":                         "hostname",
		"1.2.3.4":                          "1.2.3.4",
		"1.2.3.4:123":                      "1.2.3.4",
		"https://1.2.3.4:123":              "1.2.3.4",
		"https://1.2.3.4:123/":             "1.2.3.4",
		"https://1.2.3.4:123/foo/bar":      "1.2.3.4",
		"http://user:password@1.2.3.4":     "1.2.3.4",
		"http://user:password@1.2.3.4/foo": "1.2.3.4",
	}

	for rawURI, expected := range tests {
		actual, err := util.UrlHostNameWithoutPort(rawURI)
		assert.NoError(t, err, "for input: %s", rawURI)
		assert.Equal(t, expected, actual, "for input: %s", rawURI)
	}
}
