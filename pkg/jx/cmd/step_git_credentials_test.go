package cmd

import (
	"fmt"
	"testing"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStepGitCredentials(t *testing.T) {
	scheme1 := "https://"
	host1 := "github.com"
	user1 := "jstrachan"
	pwd1 := "lovelyLager"

	scheme2 := "http://"
	host2 := "github.beescloud.com"
	user2 := "rawlingsj"
	pwd2 := "glassOfNice"

	expected := createGitCredentialLine(scheme1, host1, user1, pwd1) +
		createGitCredentialLine(scheme2, host2, user2, pwd2)

	secretList := &corev1.SecretList{
		Items: []corev1.Secret{
			createGitSecret("foo", scheme1+host1, user1, pwd1),
			createGitSecret("bar", scheme2+host2, user2, pwd2),
		},
	}

	options := &StepGitCredentialsOptions{}

	data := options.createGitCredentialsFromSecrets(secretList)
	actual := string(data)

	assert.Equal(t, expected, actual, "generated git credentials file")

	fmt.Printf("Generated git credentials: %s\n", actual)
}

func createGitCredentialLine(scheme string, host string, user string, pwd string) string {
	return scheme + user + ":" + pwd + "@" + host + "\n"
}

func createGitSecret(name string, gitUrl string, username string, password string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				kube.AnnotationURL: gitUrl,
			},
			Labels: map[string]string{
				kube.LabelKind: kube.ValueKindGit,
			},
		},
		Data: map[string][]byte{
			kube.SecretDataUsername: []byte(username),
			kube.SecretDataPassword: []byte(password),
		},
	}
}
