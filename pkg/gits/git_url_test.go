package gits

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseGitUrlData struct {
	url          string
	hasSCM       bool
	host         string
	organisation string
	name         string
}

func TestParseGitURL(t *testing.T) {
	testCases := []parseGitUrlData{
		{
			"git://host.xz/org/repo", false, "host.xz", "org", "repo",
		},
		{
			"git://host.xz/org/repo.git", false, "host.xz", "org", "repo",
		},
		{
			"git://host.xz/org/repo.git/", false, "host.xz", "org", "repo",
		},
		{
			"git://github.com/jstrachan/npm-pipeline-test-project.git", false, "github.com", "jstrachan", "npm-pipeline-test-project",
		},
		{
			"https://github.com/fabric8io/foo.git", false, "github.com", "fabric8io", "foo",
		},
		{
			"https://github.com/fabric8io/foo", false, "github.com", "fabric8io", "foo",
		},
		{
			"git@github.com:jstrachan/npm-pipeline-test-project.git", false, "github.com", "jstrachan", "npm-pipeline-test-project",
		},
		{
			"git@github.com:bar/foo.git", false, "github.com", "bar", "foo",
		},
		{
			"git@github.com:bar/foo", false, "github.com", "bar", "foo",
		},
		{
			"bar/foo", false, "github.com", "bar", "foo",
		},
		{
			"http://test-user@auth.example.com/scm/bar/foo.git", true, "auth.example.com", "bar", "foo",
		},
	}
	for _, data := range testCases {
		info, err := ParseGitURL(data.url, data.hasSCM)
		assert.Nil(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, data.host, info.Host, "Host does not match for input %s", data.url)
		assert.Equal(t, data.organisation, info.Organisation, "Organisation does not match for input %s", data.url)
		assert.Equal(t, data.name, info.Name, "Name does not match for input %s", data.url)
	}
}
