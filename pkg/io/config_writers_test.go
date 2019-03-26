package io_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/io"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestFileConfigWriter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config auth.Config
		err    bool
	}{
		"write config to file": {
			config: auth.Config{
				Servers: []*auth.Server{
					{
						URL: "https://github.com",
						Users: []*auth.User{
							{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name:        "GitHub",
						ServiceKind: "github",
					},
				},
			},
			err: false,
		},
		"write empty config to file": {
			config: auth.Config{
				Servers: []*auth.Server{},
			},
			err: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			file, err := ioutil.TempFile("", "test-config")
			assert.NoError(t, err, "should create a temporary config file")
			defer os.Remove(file.Name())

			configWriter := io.NewFileConfigWriter(file.Name())
			err = configWriter.Write(&tc.config)
			if tc.err {
				assert.Error(t, err, "should write config into a file with an error")
			} else {
				assert.NoError(t, err, "should write config into a file without error")

				configReader := io.NewFileConfigReader(file.Name())
				config, err := configReader.Read()
				assert.NoError(t, err, "should read the written file without error")
				if config == nil {
					t.Fatal("should read a config object which is not nil")
				}
				assert.Equal(t, tc.config, *config)
			}
		})
	}
}

func TestKubeSecretsConfigWriter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		namespace string
		config    *auth.Config
		setup     func(t *testing.T, client kubernetes.Interface, ns string)
		err       bool
		want      []*corev1.Secret
	}{
		"store config into kubernetes secret": {
			namespace: "test",
			config: &auth.Config{
				Servers: []*auth.Server{
					{
						URL: "https://github.com",
						Users: []*auth.User{
							{
								Username: "test1",
								ApiToken: "test1",
								Kind:     auth.UserKindPipeline,
							},
							{
								Username: "test2",
								ApiToken: "test2",
								Kind:     auth.UserKindLocal,
							},
						},
						Name:        "GitHub",
						Kind:        auth.ServerKindGit,
						ServiceKind: auth.ServiceKindGithub,
					},
				},
			},
			err: false,
			want: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							kube.LabelCredentialsType: kube.ValueCredentialTypeUsernamePassword,
							kube.LabelCreatedBy:       kube.ValueCreatedByJX,
							kube.LabelKind:            "git",
							kube.LabelServiceKind:     "github",
						},
						Annotations: map[string]string{
							kube.AnnotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							kube.AnnotationURL:                    "https://github.com",
							kube.AnnotationName:                   "GitHub",
						},
					},
					Data: map[string][]byte{
						"username": []byte("test1"),
						"password": []byte("test1"),
					},
				},
			},
		},
		"store config into multiple kubernetes secrets": {
			namespace: "test",
			config: &auth.Config{
				Servers: []*auth.Server{
					{
						URL: "https://github.com",
						Users: []*auth.User{
							{
								Username: "test1",
								ApiToken: "test1",
								Kind:     auth.UserKindPipeline,
							},
							{
								Username: "test2",
								ApiToken: "test2",
								Kind:     auth.UserKindLocal,
							},
						},
						Name:        "GitHub",
						Kind:        auth.ServerKindGit,
						ServiceKind: auth.ServiceKindGithub,
					},
					{
						URL: "https://gitlab.com",
						Users: []*auth.User{
							{
								Username: "test1",
								ApiToken: "test1",
								Kind:     auth.UserKindPipeline,
							},
							{
								Username: "test2",
								ApiToken: "test2",
								Kind:     auth.UserKindLocal,
							},
						},
						Name:        "Gitlab",
						Kind:        auth.ServerKindGit,
						ServiceKind: auth.ServiceKindGitlab,
					},
				},
			},
			err: false,
			want: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							kube.LabelCredentialsType: kube.ValueCredentialTypeUsernamePassword,
							kube.LabelCreatedBy:       kube.ValueCreatedByJX,
							kube.LabelKind:            "git",
							kube.LabelServiceKind:     "github",
						},
						Annotations: map[string]string{
							kube.AnnotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							kube.AnnotationURL:                    "https://github.com",
							kube.AnnotationName:                   "GitHub",
						},
					},
					Data: map[string][]byte{
						"username": []byte("test1"),
						"password": []byte("test1"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-gitlab-gitlab",
						Namespace: "test",
						Labels: map[string]string{
							kube.LabelCredentialsType: kube.ValueCredentialTypeUsernamePassword,
							kube.LabelCreatedBy:       kube.ValueCreatedByJX,
							kube.LabelKind:            "git",
							kube.LabelServiceKind:     "gitlab",
						},
						Annotations: map[string]string{
							kube.AnnotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://gitlab.com"),
							kube.AnnotationURL:                    "https://gitlab.com",
							kube.AnnotationName:                   "Gitlab",
						},
					},
					Data: map[string][]byte{
						"username": []byte("test1"),
						"password": []byte("test1"),
					},
				},
			},
		},
		"store config with invalid user into kubernetes secret": {
			namespace: "test",
			config: &auth.Config{
				Servers: []*auth.Server{
					{
						URL:         "https://github.com",
						Users:       []*auth.User{},
						Name:        "GitHub",
						Kind:        auth.ServerKindGit,
						ServiceKind: auth.ServiceKindGithub,
					},
				},
			},
			err: true,
		},
		"update config into kubernetes secret": {
			namespace: "test",
			setup: func(t *testing.T, client kubernetes.Interface, ns string) {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							kube.LabelCredentialsType: kube.ValueCredentialTypeUsernamePassword,
							kube.LabelCreatedBy:       kube.ValueCreatedByJX,
							kube.LabelKind:            "git",
							kube.LabelServiceKind:     "github",
						},
						Annotations: map[string]string{
							kube.AnnotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							kube.AnnotationURL:                    "https://github.com",
							kube.AnnotationName:                   "GitHub",
						},
					},
					Data: map[string][]byte{
						"username": []byte("test"),
						"password": []byte("test"),
					},
				}

				_, err := client.CoreV1().Secrets(ns).Create(secret)
				assert.NoError(t, err, "should setup a secret without error")
			},
			config: &auth.Config{
				Servers: []*auth.Server{
					{
						URL: "https://github.com",
						Users: []*auth.User{
							{
								Username: "test1",
								ApiToken: "test1",
								Kind:     auth.UserKindPipeline,
							},
							{
								Username: "test2",
								ApiToken: "test2",
								Kind:     auth.UserKindLocal,
							},
						},
						Name:        "GitHub",
						Kind:        auth.ServerKindGit,
						ServiceKind: auth.ServiceKindGithub,
					},
				},
			},
			err: false,
			want: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							kube.LabelCredentialsType: kube.ValueCredentialTypeUsernamePassword,
							kube.LabelCreatedBy:       kube.ValueCreatedByJX,
							kube.LabelKind:            "git",
							kube.LabelServiceKind:     "github",
						},
						Annotations: map[string]string{
							kube.AnnotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							kube.AnnotationURL:                    "https://github.com",
							kube.AnnotationName:                   "GitHub",
						},
					},
					Data: map[string][]byte{
						"username": []byte("test1"),
						"password": []byte("test1"),
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := k8sfake.NewSimpleClientset()
			if tc.setup != nil {
				tc.setup(t, client, tc.namespace)
			}
			configWriter := io.NewKubeSecretsConfigWriter(client, tc.namespace)
			err := configWriter.Write(tc.config)
			if tc.err {
				assert.Error(t, err, "should generate an error when writing the config")
			} else {
				assert.NoError(t, err, "should not generate an error when writing the confiig")
				for _, wantSecret := range tc.want {
					gotSecret, err := client.CoreV1().Secrets(tc.namespace).Get(wantSecret.Name, metav1.GetOptions{})
					assert.NoErrorf(t, err, "should find secret '%s'", wantSecret.Name)
					if gotSecret == nil {
						t.Fatalf("created secret '%s' should not be nil", wantSecret.Name)
					}
					assert.Equal(t, *wantSecret, *gotSecret)
				}
			}

		})
	}

}
