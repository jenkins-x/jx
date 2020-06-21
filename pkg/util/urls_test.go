// +build unit

package util_test

import (
	"strings"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/stretchr/testify/assert"
)

const testSvcURL = "https://jx-test.com"

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

func TestSanitizeURL(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"http://test.com":                 "http://test.com",
		"http://user:test@github.com":     "http://github.com",
		"https://user:test@github.com":    "https://github.com",
		"https://user:@github.com":        "https://github.com",
		"https://:pass@github.com":        "https://github.com",
		"git@github.com:jenkins-x/jx.git": "git@github.com:jenkins-x/jx.git",
		"invalid/url":                     "invalid/url",
	}

	for test, expected := range tests {
		t.Run(test, func(t *testing.T) {
			actual := util.SanitizeURL(test)
			assert.Equal(t, expected, actual, "for url: %s", test)
		})
	}
}

func TestIsURL(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"":                 false,
		"/a/b/c":           false,
		"http//test.com":   false,
		"http://test.com":  true,
		"https://test.com": true,
	}

	for test, expected := range tests {
		t.Run(test, func(t *testing.T) {
			actual := util.IsValidUrl(test)
			assert.Equal(t, expected, actual, "%s", test)
		})
	}
}

func TestURLToHostName(t *testing.T) {
	cases := []struct {
		desc string
		in   string
		out  string
	}{
		{"service url is empty", "", ""},
		{"service url is empty", "testing urls", ""},
		{"service url is malformed", "cache_object:foo/bar", "parse cache_object:foo/bar: first path segment in URL cannot contain colon"},
		{"service url is valid", testSvcURL, strings.Replace(testSvcURL, "https://", "", -1)},
	}

	for _, v := range cases {
		url := util.URLToHostName(v.in)
		assert.Equal(t, url, v.out)
	}
}
