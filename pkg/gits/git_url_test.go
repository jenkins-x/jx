package gits_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/stretchr/testify/assert"
)

type parseGitURLData struct {
	url          string
	host         string
	organisation string
	name         string
}

func TestParseGitURL(t *testing.T) {
	t.Parallel()
	testCases := []parseGitURLData{
		{
			"git://host.xz/org/repo", "host.xz", "org", "repo",
		},
		{
			"git://host.xz/org/repo.git", "host.xz", "org", "repo",
		},
		{
			"git://host.xz/org/repo.git/", "host.xz", "org", "repo",
		},
		{
			"git://github.com/jstrachan/npm-pipeline-test-project.git", "github.com", "jstrachan", "npm-pipeline-test-project",
		},
		{
			"https://github.com/fabric8io/foo.git", "github.com", "fabric8io", "foo",
		},
		{
			"https://github.com/fabric8io/foo", "github.com", "fabric8io", "foo",
		},
		{
			"git@github.com:jstrachan/npm-pipeline-test-project.git", "github.com", "jstrachan", "npm-pipeline-test-project",
		},
		{
			"git@github.com:bar/foo.git", "github.com", "bar", "foo",
		},
		{
			"git@github.com:bar/foo", "github.com", "bar", "foo",
		},
		{
			"bar/foo", "github.com", "bar", "foo",
		},
		{
			"http://test-user@auth.example.com/scm/bar/foo.git", "auth.example.com", "bar", "foo",
		},
		{
			"https://bitbucketserver.com/projects/myproject/repos/foo/pull-requests/1", "bitbucketserver.com", "myproject", "foo",
		},
	}
	for _, data := range testCases {
		info, err := gits.ParseGitURL(data.url)
		assert.Nil(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, data.host, info.Host, "Host does not match for input %s", data.url)
		assert.Equal(t, data.organisation, info.Organisation, "Organisation does not match for input %s", data.url)
		assert.Equal(t, data.name, info.Name, "Name does not match for input %s", data.url)
	}
}

func TestSaasKind(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		gitUrl string
		kind   string
	}{
		"GitHub": {
			gitUrl: "https://github.com/test",
			kind:   gits.KindGitHub,
		},
		"GitHub Enterprise": {
			gitUrl: "https://github.test.com",
			kind:   gits.KindGitHub,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			k := gits.SaasGitKind(tc.gitUrl)
			assert.Equal(t, tc.kind, k)
		})
	}
}
