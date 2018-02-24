package prompt

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrompt(t *testing.T) {
	var stdout bytes.Buffer

	type test struct {
		userInput string
		scenario  []struct {
			inputs []string
			err    error
		}
		expected map[string]string
	}

	tests := []test{

		// Mandatory parameters only
		{
			"HEAD\nHEAD~2\n/home/project\nq\n",
			[]struct {
				inputs []string
				err    error
			}{
				{[]string{"HEAD"}, nil},
				{[]string{"HEAD~2"}, nil},
				{[]string{"/home/project"}, nil},
				{[]string{"q"}, nil},
			},
			map[string]string{
				"CHYLE_GIT_REFERENCE_FROM":  "HEAD",
				"CHYLE_GIT_REFERENCE_TO":    "HEAD~2",
				"CHYLE_GIT_REPOSITORY_PATH": "/home/project",
			},
		},

		// Matchers
		{
			"HEAD\nHEAD~2\n/home/project\n\n999\n1\n1\nwhatever\nregular\n2\ntest.**\ntest.*\n3\njohn.**\njohn.*\n4\nsam.**\nsam.*\nm\nq\n",
			[]struct {
				inputs []string
				err    error
			}{
				{[]string{"HEAD"}, nil},
				{[]string{"HEAD~2"}, nil},
				{[]string{"/home/project"}, nil},
				{[]string{""}, fmt.Errorf("No value given")},
				{[]string{"999"}, fmt.Errorf("This choice doesn't exist")},
				{[]string{"1"}, nil},
				{[]string{"1"}, nil},
				{[]string{"whatever"}, fmt.Errorf(`Must be "regular" or "merge"`)},
				{[]string{"regular"}, nil},
				{[]string{"2"}, nil},
				{[]string{"test.**"}, fmt.Errorf("\"test.**\" is an invalid regexp : error parsing regexp: invalid nested repetition operator: `**`")},
				{[]string{"test.*"}, nil},
				{[]string{"3"}, nil},
				{[]string{"john.**"}, fmt.Errorf("\"john.**\" is an invalid regexp : error parsing regexp: invalid nested repetition operator: `**`")},
				{[]string{"john.*"}, nil},
				{[]string{"4"}, nil},
				{[]string{"sam.**"}, fmt.Errorf("\"sam.**\" is an invalid regexp : error parsing regexp: invalid nested repetition operator: `**`")},
				{[]string{"sam.*"}, nil},
				{[]string{"m"}, nil},
				{[]string{"q"}, nil},
			},
			map[string]string{
				"CHYLE_GIT_REFERENCE_FROM":  "HEAD",
				"CHYLE_GIT_REFERENCE_TO":    "HEAD~2",
				"CHYLE_GIT_REPOSITORY_PATH": "/home/project",
				"CHYLE_MATCHERS_MESSAGE":    "test.*",
				"CHYLE_MATCHERS_TYPE":       "regular",
				"CHYLE_MATCHERS_COMMITTER":  "john.*",
				"CHYLE_MATCHERS_AUTHOR":     "sam.*",
			},
		},

		// Extractors
		{
			"HEAD\nHEAD~2\n/home/project\n2\n\nwhatever\nid\nidParsed\n#\\d++\n#\\d+\nq\n",
			[]struct {
				inputs []string
				err    error
			}{
				{[]string{"HEAD"}, nil},
				{[]string{"HEAD~2"}, nil},
				{[]string{"/home/project"}, nil},
				{[]string{"2"}, nil},
				{[]string{""}, fmt.Errorf(`Must be one of [id authorName authorEmail authorDate committerName committerEmail committerMessage type]`)},
				{[]string{"whatever"}, fmt.Errorf(`Must be one of [id authorName authorEmail authorDate committerName committerEmail committerMessage type]`)},
				{[]string{"id"}, nil},
				{[]string{"idParsed"}, nil},
				{[]string{"#\\d++"}, fmt.Errorf("\"#\\d++\" is an invalid regexp : error parsing regexp: invalid nested repetition operator: `++`")},
				{[]string{"#\\d+"}, nil},
				{[]string{"q"}, nil},
			},
			map[string]string{
				"CHYLE_GIT_REFERENCE_FROM":   "HEAD",
				"CHYLE_GIT_REFERENCE_TO":     "HEAD~2",
				"CHYLE_GIT_REPOSITORY_PATH":  "/home/project",
				"CHYLE_EXTRACTORS_0_DESTKEY": "idParsed",
				"CHYLE_EXTRACTORS_0_ORIGKEY": "id",
				"CHYLE_EXTRACTORS_0_REG":     "#\\d+",
			},
		},

		// Decorators
		{
			"HEAD\nHEAD~2\n/home/project\n3\n1\nmessage\nid\n#\\d++\n#\\d+\ntest\nhttp://test.com\n=@eTN#d0t:x4TgKE|XJ531!H<n0rJH\nobjectId\nfields.id\n1\ndate\nfields.date\nm\n3\n2\nmessage\n#\\d++\n#\\d+\nhttp://api.jira.com\nuser\npassword\nobjectId\nfields.id\n1\ndate\nfields.date\nm\n3\n3\nmessage\n#\\d++\n#\\d+\nd41d8cd98f00b204e9800998ecf8427e\nuser\nobjectId\nfields.id\n1\ndate\nfields.date\nm\n3\n4\necho\nmessage\nid\n4\necho\nmessage\nfield\nm\n3\n5\nTEST\ntest\n5\nfoo\nbar\nq\n",
			[]struct {
				inputs []string
				err    error
			}{
				{[]string{"HEAD"}, nil},
				{[]string{"HEAD~2"}, nil},
				{[]string{"/home/project"}, nil},
				{[]string{"3"}, nil},
				{[]string{"1"}, nil},
				{[]string{"message"}, nil},
				{[]string{"id"}, nil},
				{[]string{"#\\d++"}, fmt.Errorf("\"#\\d++\" is an invalid regexp : error parsing regexp: invalid nested repetition operator: `++`")},
				{[]string{"#\\d+"}, nil},
				{[]string{"test"}, fmt.Errorf(`"test" must be a valid URL`)},
				{[]string{"http://test.com"}, nil},
				{[]string{"=@eTN#d0t:x4TgKE|XJ531!H<n0rJH"}, nil},
				{[]string{"objectId"}, nil},
				{[]string{"fields.id"}, nil},
				{[]string{"1"}, nil},
				{[]string{"date"}, nil},
				{[]string{"fields.date"}, nil},
				{[]string{"m"}, nil},
				{[]string{"3"}, nil},
				{[]string{"2"}, nil},
				{[]string{"message"}, nil},
				{[]string{"#\\d++"}, fmt.Errorf("\"#\\d++\" is an invalid regexp : error parsing regexp: invalid nested repetition operator: `++`")},
				{[]string{"#\\d+"}, nil},
				{[]string{"http://api.jira.com"}, nil},
				{[]string{"user"}, nil},
				{[]string{"password"}, nil},
				{[]string{"objectId"}, nil},
				{[]string{"fields.id"}, nil},
				{[]string{"1"}, nil},
				{[]string{"date"}, nil},
				{[]string{"fields.date"}, nil},
				{[]string{"m"}, nil},
				{[]string{"3"}, nil},
				{[]string{"3"}, nil},
				{[]string{"message"}, nil},
				{[]string{"#\\d++"}, fmt.Errorf("\"#\\d++\" is an invalid regexp : error parsing regexp: invalid nested repetition operator: `++`")},
				{[]string{"#\\d+"}, nil},
				{[]string{"d41d8cd98f00b204e9800998ecf8427e"}, nil},
				{[]string{"user"}, nil},
				{[]string{"objectId"}, nil},
				{[]string{"fields.id"}, nil},
				{[]string{"1"}, nil},
				{[]string{"date"}, nil},
				{[]string{"fields.date"}, nil},
				{[]string{"m"}, nil},
				{[]string{"3"}, nil},
				{[]string{"4"}, nil},
				{[]string{"echo"}, nil},
				{[]string{"message"}, nil},
				{[]string{"id"}, nil},
				{[]string{"4"}, nil},
				{[]string{"echo"}, nil},
				{[]string{"message"}, nil},
				{[]string{"field"}, nil},
				{[]string{"m"}, nil},
				{[]string{"3"}, nil},
				{[]string{"5"}, nil},
				{[]string{"TEST"}, nil},
				{[]string{"test"}, nil},
				{[]string{"5"}, nil},
				{[]string{"foo"}, nil},
				{[]string{"bar"}, nil},
				{[]string{"q"}, nil},
			},
			map[string]string{
				"CHYLE_GIT_REFERENCE_FROM":                            "HEAD",
				"CHYLE_GIT_REFERENCE_TO":                              "HEAD~2",
				"CHYLE_GIT_REPOSITORY_PATH":                           "/home/project",
				"CHYLE_DECORATORS_CUSTOMAPIID_CREDENTIALS_TOKEN":      "=@eTN#d0t:x4TgKE|XJ531!H<n0rJH",
				"CHYLE_DECORATORS_CUSTOMAPIID_ENDPOINT_URL":           "http://test.com",
				"CHYLE_EXTRACTORS_CUSTOMAPIID_DESTKEY":                "id",
				"CHYLE_EXTRACTORS_CUSTOMAPIID_ORIGKEY":                "message",
				"CHYLE_EXTRACTORS_CUSTOMAPIID_REG":                    "#\\d+",
				"CHYLE_DECORATORS_CUSTOMAPIID_KEYS_0_DESTKEY":         "objectId",
				"CHYLE_DECORATORS_CUSTOMAPIID_KEYS_0_FIELD":           "fields.id",
				"CHYLE_DECORATORS_CUSTOMAPIID_KEYS_1_DESTKEY":         "date",
				"CHYLE_DECORATORS_CUSTOMAPIID_KEYS_1_FIELD":           "fields.date",
				"CHYLE_EXTRACTORS_JIRAISSUEID_ORIGKEY":                "message",
				"CHYLE_EXTRACTORS_JIRAISSUEID_DESTKEY":                "jiraIssueId",
				"CHYLE_EXTRACTORS_JIRAISSUEID_REG":                    "#\\d+",
				"CHYLE_DECORATORS_JIRAISSUE_ENDPOINT_URL":             "http://api.jira.com",
				"CHYLE_DECORATORS_JIRAISSUE_CREDENTIALS_USERNAME":     "user",
				"CHYLE_DECORATORS_JIRAISSUE_CREDENTIALS_PASSWORD":     "password",
				"CHYLE_DECORATORS_JIRAISSUE_KEYS_0_DESTKEY":           "objectId",
				"CHYLE_DECORATORS_JIRAISSUE_KEYS_0_FIELD":             "fields.id",
				"CHYLE_DECORATORS_JIRAISSUE_KEYS_1_DESTKEY":           "date",
				"CHYLE_DECORATORS_JIRAISSUE_KEYS_1_FIELD":             "fields.date",
				"CHYLE_EXTRACTORS_GITHUBISSUEID_ORIGKEY":              "message",
				"CHYLE_EXTRACTORS_GITHUBISSUEID_DESTKEY":              "githubIssueId",
				"CHYLE_EXTRACTORS_GITHUBISSUEID_REG":                  "#\\d+",
				"CHYLE_DECORATORS_GITHUBISSUE_CREDENTIALS_OAUTHTOKEN": "d41d8cd98f00b204e9800998ecf8427e",
				"CHYLE_DECORATORS_GITHUBISSUE_CREDENTIALS_OWNER":      "user",
				"CHYLE_DECORATORS_GITHUBISSUE_KEYS_0_DESTKEY":         "objectId",
				"CHYLE_DECORATORS_GITHUBISSUE_KEYS_0_FIELD":           "fields.id",
				"CHYLE_DECORATORS_GITHUBISSUE_KEYS_1_DESTKEY":         "date",
				"CHYLE_DECORATORS_GITHUBISSUE_KEYS_1_FIELD":           "fields.date",
				"CHYLE_DECORATORS_SHELL_0_COMMAND":                    "echo",
				"CHYLE_DECORATORS_SHELL_0_ORIGKEY":                    "message",
				"CHYLE_DECORATORS_SHELL_0_DESTKEY":                    "id",
				"CHYLE_DECORATORS_SHELL_1_COMMAND":                    "echo",
				"CHYLE_DECORATORS_SHELL_1_ORIGKEY":                    "message",
				"CHYLE_DECORATORS_SHELL_1_DESTKEY":                    "field",
				"CHYLE_DECORATORS_ENV_0_VARNAME":                      "TEST",
				"CHYLE_DECORATORS_ENV_0_DESTKEY":                      "test",
				"CHYLE_DECORATORS_ENV_1_VARNAME":                      "foo",
				"CHYLE_DECORATORS_ENV_1_DESTKEY":                      "bar",
			},
		},

		// Senders
		{
			"HEAD\nHEAD~2\n/home/project\n4\n1\njso\njson\n1\ntemplate\n{{{\n{{.}}\n2\nd41d8cd98f00b204e9800998ecf8427e\nuser\nrepository\nwhatever\nfalse\nRelease 1\nwhatever\nfalse\nv1.0.0\nmaster\n{{{\n{{.}}\nwhatever\nfalse\n3\nd41d8cd98f00b204e9800998ecf8427e\ntest\nhttp://test.com\nq\n",
			[]struct {
				inputs []string
				err    error
			}{
				{[]string{"HEAD"}, nil},
				{[]string{"HEAD~2"}, nil},
				{[]string{"/home/project"}, nil},
				{[]string{"4"}, nil},
				{[]string{"1"}, nil},
				{[]string{"jso"}, fmt.Errorf(`"jso" is not a valid format, it must be either "json" or "template"`)},
				{[]string{"json"}, nil},
				{[]string{"1"}, nil},
				{[]string{"template"}, nil},
				{[]string{"{{{"}, fmt.Errorf(`"{{{" is an invalid template : template: template:1: unexpected "{" in command`)},
				{[]string{"{{.}}"}, nil},
				{[]string{"2"}, nil},
				{[]string{"d41d8cd98f00b204e9800998ecf8427e"}, nil},
				{[]string{"user"}, nil},
				{[]string{"repository"}, nil},
				{[]string{"whatever"}, fmt.Errorf(`"whatever" must be true or false`)},
				{[]string{"false"}, nil},
				{[]string{"Release 1"}, nil},
				{[]string{"whatever"}, fmt.Errorf(`"whatever" must be true or false`)},
				{[]string{"false"}, nil},
				{[]string{"v1.0.0"}, nil},
				{[]string{"master"}, nil},
				{[]string{"{{{"}, fmt.Errorf(`"{{{" is an invalid template : template: template:1: unexpected "{" in command`)},
				{[]string{"{{.}}"}, nil},
				{[]string{"whatever"}, fmt.Errorf(`"whatever" must be true or false`)},
				{[]string{"false"}, nil},
				{[]string{"3"}, nil},
				{[]string{"d41d8cd98f00b204e9800998ecf8427e"}, nil},
				{[]string{"test"}, fmt.Errorf(`"test" must be a valid URL`)},
				{[]string{"http://test.com"}, nil},
				{[]string{"q"}, nil},
			},
			map[string]string{
				"CHYLE_GIT_REFERENCE_FROM":                            "HEAD",
				"CHYLE_GIT_REFERENCE_TO":                              "HEAD~2",
				"CHYLE_GIT_REPOSITORY_PATH":                           "/home/project",
				"CHYLE_SENDERS_STDOUT_FORMAT":                         "template",
				"CHYLE_SENDERS_STDOUT_TEMPLATE":                       "{{.}}",
				"CHYLE_SENDERS_GITHUBRELEASE_CREDENTIALS_OAUTHTOKEN":  "d41d8cd98f00b204e9800998ecf8427e",
				"CHYLE_SENDERS_GITHUBRELEASE_CREDENTIALS_OWNER":       "user",
				"CHYLE_SENDERS_GITHUBRELEASE_REPOSITORY_NAME":         "repository",
				"CHYLE_SENDERS_GITHUBRELEASE_RELEASE_DRAFT":           "false",
				"CHYLE_SENDERS_GITHUBRELEASE_RELEASE_NAME":            "Release 1",
				"CHYLE_SENDERS_GITHUBRELEASE_RELEASE_PRERELEASE":      "false",
				"CHYLE_SENDERS_GITHUBRELEASE_RELEASE_TAGNAME":         "v1.0.0",
				"CHYLE_SENDERS_GITHUBRELEASE_RELEASE_TARGETCOMMITISH": "master",
				"CHYLE_SENDERS_GITHUBRELEASE_RELEASE_TEMPLATE":        "{{.}}",
				"CHYLE_SENDERS_GITHUBRELEASE_RELEASE_UPDATE":          "false",
				"CHYLE_SENDERS_CUSTOMAPI_CREDENTIALS_TOKEN":           "d41d8cd98f00b204e9800998ecf8427e",
				"CHYLE_SENDERS_CUSTOMAPI_ENDPOINT_URL":                "http://test.com",
			},
		},
	}

	for _, test := range tests {
		buf := test.userInput

		p := New(bytes.NewBufferString(buf), &stdout)

		envs := p.Run()

		assert.Equal(t, test.expected, map[string]string(envs))

		for i, s := range test.scenario {
			if i+1 > len(p.prompts.Scenario()) {
				t.Fatal("Scenario doesn't match expected one")
			}

			assert.Equal(t, s.inputs, p.prompts.Scenario()[i].Inputs())
			assert.Equal(t, s.err, p.prompts.Scenario()[i].Error())
		}
	}
}
