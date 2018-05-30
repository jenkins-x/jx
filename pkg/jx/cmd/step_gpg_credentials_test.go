package cmd

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStepGPGCredentials(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-step-gpg")
	assert.NoError(t, err)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			"pubring.gpg":     []byte("Pubring"),
			"sec-jenkins.gpg": []byte("sec jenkins"),
			"secring.gpg":     []byte("secring"),
			"trustdb.gpg":     []byte("trustdb"),
		},
	}

	options := &StepGpgCredentialsOptions{
		OutputDir: tempDir,
	}

	err = options.GenerateGpgFiles(secret)
	assert.NoError(t, err)

	assertFileExists(t, filepath.Join(tempDir, "pubring.gpg"))
	assertFileExists(t, filepath.Join(tempDir, "sec-jenkins.gpg"))
	assertFileExists(t, filepath.Join(tempDir, "secring.gpg"))
	assertFileExists(t, filepath.Join(tempDir, "trustdb.gpg"))
}
