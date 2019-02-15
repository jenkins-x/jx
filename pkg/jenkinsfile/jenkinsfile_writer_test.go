package jenkinsfile_test

import (
	"github.com/jenkins-x/jx/pkg/jenkinsfile"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJenkinsfileWriter(t *testing.T) {
	expectedValue := `container('maven') {
  dir('/foo/bar') {
    sh "ls -al"
    sh "mvn deploy"
  }
}
`
	writer := jenkinsfile.NewWriter(0)

	statements := []*jenkinsfile.Statement{
		{
			Function:  "container",
			Arguments: []string{"maven"},
			Children: []*jenkinsfile.Statement{
				{
					Function:  "dir",
					Arguments: []string{"/foo/bar"},
					Children: []*jenkinsfile.Statement{
						{
							Statement: "sh \"ls -al\"",
						},
					},
				},
			},
		},
		{
			Function:  "container",
			Arguments: []string{"maven"},
			Children: []*jenkinsfile.Statement{
				{
					Function:  "dir",
					Arguments: []string{"/foo/bar"},
					Children: []*jenkinsfile.Statement{
						{
							Statement: "sh \"mvn deploy\"",
						},
					},
				},
			},
		},
	}
	writer.Write(statements)
	text := writer.String()

	assert.Equal(t, expectedValue, text, "for statements %#v", statements)
}
