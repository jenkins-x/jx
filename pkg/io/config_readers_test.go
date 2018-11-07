package io_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/io"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestFileConfigReader(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config     string
		createFile bool
		err        bool
		want       auth.Config
	}{
		"read config from file": {
			config: `
servers:
- url: https://github.com
  users:
  - username: test
    apitoken: test
    bearertoken: ""
  name: GitHub
  kind: github`,
			createFile: true,
			err:        false,
			want: auth.Config{
				Servers: []*auth.Server{
					&auth.Server{
						URL: "https://github.com",
						Users: []*auth.User{
							&auth.User{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name: "GitHub",
						Kind: "github",
					},
				},
			},
		},
		"read config from empty file": {
			config:     "",
			createFile: true,
			err:        false,
			want:       auth.Config{},
		},
		"read config from a file which does not exist": {
			config:     "",
			createFile: false,
			err:        true,
			want:       auth.Config{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			configFile := "test"
			if tc.createFile {
				file, err := ioutil.TempFile("", "test-config")
				assert.NoError(t, err, "should create a temporary config file")
				defer os.Remove(file.Name())
				_, err = file.Write([]byte(tc.config))
				assert.NoError(t, err, "should write the test config into file")
				configFile = file.Name()
			}
			configReader := io.NewFileConfigReader(configFile)
			config, err := configReader.Read()
			if tc.err {
				assert.Error(t, err, "should read config from file with an error")
			} else {
				assert.NoError(t, err, "should read config from file without error")
				if config == nil {
					t.Fatal("should read a config object which is not nil")
				}
				assert.Equal(t, tc.want, *config)
			}
		})
	}
}

func setEnvs(t *testing.T, envs map[string]string) {
	err := util.RestoreEnviron(envs)
	assert.NoError(t, err, "should set the environment variables")
}

func cleanEnvs(t *testing.T, envs []string) {
	_, err := util.GetAndCleanEnviron(envs)
	assert.NoError(t, err, "shuold clean the environment variables")
}

func TestEnvConfigReader(t *testing.T) {
	t.Parallel()

	const prefix = "TEST"
	tests := map[string]struct {
		prefix          string
		serverRetriever io.ServerRetrieverFn
		setup           func(t *testing.T)
		cleanup         func(t *testing.T)
		err             bool
		want            auth.Config
	}{
		"read config from environment variables": {
			prefix: prefix,
			serverRetriever: func() (string, string, auth.ServerKind) {
				return "GitHub", "https://github.com", auth.ServerKindGithub
			},
			setup: func(t *testing.T) {
				setEnvs(t, map[string]string{
					auth.UsernameEnv(prefix): "test",
					auth.ApiTokenEnv(prefix): "test",
				})
			},
			cleanup: func(t *testing.T) {
				cleanEnvs(t, []string{
					auth.UsernameEnv(prefix),
					auth.ApiTokenEnv(prefix),
				})
			},
			want: auth.Config{
				Servers: []*auth.Server{
					&auth.Server{
						URL: "https://github.com",
						Users: []*auth.User{
							&auth.User{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name: "GitHub",
						Kind: "github",
					},
				},
			},
			err: false,
		},
		"read config from empty environment variables": {
			prefix: prefix,
			serverRetriever: func() (string, string, auth.ServerKind) {
				return "GitHub", "https://github.com", auth.ServerKindGithub
			},
			want: auth.Config{},
			err:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			configReader := io.NewEnvConfigReader(tc.prefix, tc.serverRetriever)
			config, err := configReader.Read()
			if tc.err {
				assert.Error(t, err, "should read config from env with error")
			} else {
				assert.NoError(t, err, "should read config from env with error")
				if config == nil {
					t.Fatal("should read a config object which is not nil")
				}
				assert.Equal(t, tc.want, *config)
			}
			if tc.cleanup != nil {
				tc.cleanup(t)
			}
		})
	}
}

func secret(name string, kind string, serviceKind auth.ServerKind, serviceName string, url string, username string, password string) *corev1.Secret {
	labels := map[string]string{}
	if kind != "" {
		labels[kube.LabelKind] = kind
	}
	if serviceKind != "" {
		labels[kube.LabelServiceKind] = string(serviceKind)
	}
	annotations := map[string]string{
		kube.AnnotationName: serviceName,
	}
	if url != "" {
		annotations[kube.AnnotationURL] = url
	}
	data := map[string][]byte{}
	if username != "" {
		data[kube.SecretDataUsername] = []byte(username)
	}
	if password != "" {
		data[kube.SecretDataPassword] = []byte(password)
	}
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: data,
	}
	return s
}

func TestKubeSecretsConfigReader(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name       string
		namespace  string
		kind       string
		serverKind auth.ServerKind
		url        string
		username   string
		password   string
		want       auth.Config
		err        bool
	}{
		"read config from k8s secret": {
			name:       "GitHub",
			namespace:  "test",
			kind:       "git",
			serverKind: auth.ServerKindGithub,
			url:        "https://github.com",
			username:   "test",
			password:   "test",
			want: auth.Config{
				Servers: []*auth.Server{
					&auth.Server{
						URL: "https://github.com",
						Users: []*auth.User{
							&auth.User{
								Username: "test",
								ApiToken: "test",
								Kind:     auth.UserKindPipeline,
							},
						},
						Name: "GitHub",
						Kind: "github",
					},
				},
			},
			err: false,
		},
		"read config from k8s secret without service kind": {
			name:       "GitHub",
			namespace:  "test",
			kind:       "git",
			serverKind: "",
			url:        "https://github.com",
			username:   "test",
			password:   "test",
			want: auth.Config{
				Servers: []*auth.Server{
					&auth.Server{
						URL: "https://github.com",
						Users: []*auth.User{
							&auth.User{
								Username: "test",
								ApiToken: "test",
								Kind:     auth.UserKindPipeline,
							},
						},
						Name: "GitHub",
						Kind: "",
					},
				},
			},
			err: false,
		},
		"read config from k8s secret without kind": {
			name:       "GitHub",
			namespace:  "test",
			kind:       "",
			serverKind: auth.ServerKindGithub,
			url:        "https://github.com",
			username:   "test",
			password:   "test",
			want: auth.Config{
				Servers: []*auth.Server{
					&auth.Server{
						URL: "https://github.com",
						Users: []*auth.User{
							&auth.User{
								Username: "test",
								ApiToken: "test",
								Kind:     auth.UserKindPipeline,
							},
						},
						Name: "GitHub",
						Kind: "github",
					},
				},
			},
			err: false,
		},
		"read config from k8s secret without kind labels": {
			name:       "GitHub",
			namespace:  "test",
			kind:       "",
			serverKind: "",
			url:        "https://github.com",
			username:   "test",
			password:   "test",
			want:       auth.Config{},
			err:        false,
		},
		"read config from k8s secret without username": {
			name:       "GitHub",
			namespace:  "test",
			kind:       "git",
			serverKind: "github",
			url:        "https://github.com",
			username:   "",
			password:   "test",
			want:       auth.Config{},
			err:        false,
		},
		"read config from k8s secret without password": {
			name:       "GitHub",
			namespace:  "test",
			kind:       "git",
			serverKind: "github",
			url:        "https://github.com",
			username:   "test",
			password:   "",
			want:       auth.Config{},
			err:        false,
		},
		"read config from k8s secret without URL": {
			name:       "GitHub",
			namespace:  "test",
			kind:       "git",
			serverKind: "github",
			url:        "",
			username:   "test",
			password:   "test",
			want:       auth.Config{},
			err:        false,
		},
		"read config from k8s secret without name": {
			name:       "",
			namespace:  "test",
			kind:       "git",
			serverKind: auth.ServerKindGithub,
			url:        "https://github.com",
			username:   "test",
			password:   "test",
			want: auth.Config{
				Servers: []*auth.Server{
					&auth.Server{
						URL: "https://github.com",
						Users: []*auth.User{
							&auth.User{
								Username: "test",
								ApiToken: "test",
								Kind:     auth.UserKindPipeline,
							},
						},
						Name: "",
						Kind: "github",
					},
				},
			},
			err: false,
		},
		"read config from k8s secret without labels and annotations": {
			name:       "",
			namespace:  "test",
			kind:       "",
			serverKind: "",
			url:        "",
			username:   "",
			password:   "",
			want:       auth.Config{},
			err:        false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := k8sfake.NewSimpleClientset()
			const secretName = "config-test"
			secret := secret(secretName, tc.kind, tc.serverKind, tc.name, tc.url, tc.username, tc.password)
			_, err := client.CoreV1().Secrets(tc.namespace).Create(secret)
			assert.NoError(t, err, "should create secret without error")

			configReader := io.NewKubeSecretsConfigReader(client, tc.namespace, tc.kind, tc.serverKind)
			config, err := configReader.Read()
			if tc.err {
				assert.Error(t, err, "should read config from secrete with error")
			} else {
				assert.NoError(t, err, "should read config from secret without error")
				if config == nil {
					t.Fatal("should read a config object which is not nil")
				}
				assert.Equal(t, tc.want, *config)
			}
			err = client.CoreV1().Secrets(tc.namespace).Delete(secretName, &metav1.DeleteOptions{})
			assert.NoError(t, err, "should delete the secret")
		})
	}

}
