package git_test

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/step/git"
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/testkube"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestStepGitCredentials(t *testing.T) {
	t.Parallel()
	kind1 := gits.KindGitHub
	scheme1 := "https://"
	host1 := "github.com"
	user1 := "jstrachan"
	pwd1 := "lovelyLager"

	kind2 := gits.KindGitHub
	scheme2 := "http://"
	host2 := "github.beescloud.com"
	user2 := "rawlingsj"
	pwd2 := "glassOfNice"

	expected := createGitCredentialLine(scheme1, host1, user1, pwd1) +
		createGitCredentialLine(scheme2, host2, user2, pwd2)

	secretList := &corev1.SecretList{
		Items: []corev1.Secret{
			testkube.CreateTestPipelineGitSecret(kind1, "foo", scheme1+host1, user1, pwd1),
			testkube.CreateTestPipelineGitSecret(kind2, "bar", scheme2+host2, user2, pwd2),
		},
	}

	options := &git.StepGitCredentialsOptions{}

	data := options.CreateGitCredentialsFromSecrets(secretList)
	actual := string(data)

	assert.Equal(t, expected, actual, "generated git credentials file")

	tests.Debugf("Generated git credentials: %s\n", actual)
}

func createGitCredentialLine(scheme string, host string, user string, pwd string) string {
	answer := scheme + user + ":" + pwd + "@" + host + "\n"
	if scheme == "https://" {
		scheme = "http://"
	} else {
		scheme = "https://"
	}
	answer += scheme + user + ":" + pwd + "@" + host + "\n"
	return answer
}
