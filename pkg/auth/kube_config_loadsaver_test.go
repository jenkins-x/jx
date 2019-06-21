package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func secret(name string, serverKind string, serviceKind string, createLabels bool,
	serviceName string, url string, username string, password string) *corev1.Secret {
	labels := map[string]string{}
	if serverKind != "" || createLabels {
		labels[labelKind] = serverKind
	}
	if serviceKind != "" || createLabels {
		labels[labelServiceKind] = serviceKind
	}
	annotations := map[string]string{
		annotationName: serviceName,
	}
	if url != "" {
		annotations[annotationURL] = url
	}
	data := map[string][]byte{}
	if username != "" {
		data[usernameKey] = []byte(username)
	}
	if password != "" {
		data[passwordKey] = []byte(password)
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

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name         string
		namespace    string
		serverKind   string
		serviceKind  string
		createLabels bool
		url          string
		username     string
		password     string
		want         Config
		err          bool
	}{
		"load config from k8s secret": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "git",
			serviceKind:  "github",
			createLabels: true,
			url:          "https://github.com",
			username:     "test",
			password:     "test",
			want: Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name:        "GitHub",
						Kind:        "github",
						CurrentUser: "test",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
		},
		"load config from k8s secret without service kind": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "git",
			serviceKind:  "",
			createLabels: true,
			url:          "https://github.com",
			username:     "test",
			password:     "test",
			want: Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name:        "GitHub",
						Kind:        "",
						CurrentUser: "test",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
		},
		"load config from k8s secret without kind": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "",
			serviceKind:  "github",
			createLabels: true,
			url:          "https://github.com",
			username:     "test",
			password:     "test",
			want: Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name:        "GitHub",
						Kind:        "github",
						CurrentUser: "test",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
		},
		"load config from k8s secret with empty kind": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "",
			serviceKind:  "",
			createLabels: true,
			url:          "https://github.com",
			username:     "test",
			password:     "test",
			want: Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name:        "GitHub",
						Kind:        "",
						CurrentUser: "test",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
		},
		"load config from k8s secret without name": {
			name:         "",
			namespace:    "test",
			serverKind:   "git",
			serviceKind:  "github",
			createLabels: true,
			url:          "https://github.com",
			username:     "test",
			password:     "test",
			want: Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test",
								ApiToken: "test",
							},
						},
						Name:        "",
						Kind:        "github",
						CurrentUser: "test",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
		},
		"load config from k8s secret without kind labels": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "",
			serviceKind:  "",
			createLabels: false,
			url:          "https://github.com",
			username:     "test",
			password:     "test",
			want:         Config{},
			err:          false,
		},
		"load config from k8s secret without kind labels and annotations": {
			name:         "",
			namespace:    "test",
			serverKind:   "",
			serviceKind:  "",
			createLabels: false,
			url:          "",
			username:     "",
			password:     "",
			want:         Config{},
			err:          false,
		},
		"load config from k8s secret without username": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "git",
			serviceKind:  "github",
			createLabels: true,
			url:          "https://github.com",
			username:     "",
			password:     "test",
			want:         Config{},
			err:          false,
		},
		"load config from k8s secret without password": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "git",
			serviceKind:  "github",
			createLabels: true,
			url:          "https://github.com",
			username:     "test",
			password:     "",
			want:         Config{},
			err:          false,
		},
		"load config from k8s secret without URL": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "git",
			serviceKind:  "github",
			createLabels: true,
			url:          "",
			username:     "test",
			password:     "test",
			want:         Config{},
			err:          false,
		},
		"load config from k8s secret with invalid user": {
			name:         "GitHub",
			namespace:    "test",
			serverKind:   "git",
			serviceKind:  "github",
			createLabels: true,
			url:          "https://github.com",
			username:     "",
			password:     "",
			want:         Config{},
			err:          false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := k8sfake.NewSimpleClientset()
			const secretName = "config-test"
			secret := secret(secretName, tc.serverKind, tc.serviceKind, tc.createLabels,
				tc.name, tc.url, tc.username, tc.password)
			_, err := client.CoreV1().Secrets(tc.namespace).Create(secret)
			assert.NoError(t, err, "should create secret without error")

			svc := NewKubeConfigService(client, tc.namespace, tc.serverKind, tc.serviceKind)
			config, err := svc.Config()
			if tc.err {
				assert.Error(t, err, "should load config from secret with an error")
			} else {
				assert.NoError(t, err, "should load config from secret without an error")
				if config == nil {
					t.Fatal("config should not be nil")
				}
				assert.Equal(t, tc.want, *config)
			}

			err = client.CoreV1().Secrets(tc.namespace).Delete(secretName, &metav1.DeleteOptions{})
			assert.NoError(t, err, "should delete the secret")
		})
	}

}

func TestSaveConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		namespace  string
		serverKind string
		config     *Config
		setup      func(t *testing.T, client kubernetes.Interface, ns string)
		err        bool
		want       []*corev1.Secret
	}{
		"save config into kubernetes secret": {
			namespace:  "test",
			serverKind: "git",
			config: &Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test1",
								ApiToken: "test1",
							},
							{
								Username: "test2",
								ApiToken: "test2",
							},
						},
						Name:        "GitHub",
						Kind:        "github",
						CurrentUser: "test1",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
			want: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							labelCredentialsType: valueCredentialTypeUsernamePassword,
							labelCreatedBy:       valueCreatedByJX,
							labelKind:            "git",
							labelServiceKind:     "github",
						},
						Annotations: map[string]string{
							annotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							annotationURL:                    "https://github.com",
							annotationName:                   "GitHub",
						},
					},
					Data: map[string][]byte{
						"username": []byte("test1"),
						"password": []byte("test1"),
					},
				},
			},
		},
		"save config into kubernetes secret without API token": {
			namespace:  "test",
			serverKind: "git",
			config: &Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test1",
								Password: "test1",
							},
						},
						Name:        "GitHub",
						Kind:        "github",
						CurrentUser: "test1",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
			want: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							labelCredentialsType: valueCredentialTypeUsernamePassword,
							labelCreatedBy:       valueCreatedByJX,
							labelKind:            "git",
							labelServiceKind:     "github",
						},
						Annotations: map[string]string{
							annotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							annotationURL:                    "https://github.com",
							annotationName:                   "GitHub",
						},
					},
					Data: map[string][]byte{
						"username": []byte("test1"),
						"password": []byte("test1"),
					},
				},
			},
		},
		"save config into multiple kubernetes secret": {
			namespace:  "test",
			serverKind: "git",
			config: &Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test1",
								ApiToken: "test1",
							},
							{
								Username: "test2",
								ApiToken: "test2",
							},
						},
						Name:        "GitHub",
						Kind:        "github",
						CurrentUser: "test1",
					},
					{
						URL: "https://gitlab.com",
						Users: []User{
							{
								Username: "test1",
								ApiToken: "test1",
							},
							{
								Username: "test2",
								ApiToken: "test2",
							},
						},
						Name:        "GitLab",
						Kind:        "gitlab",
						CurrentUser: "test2",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
			want: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							labelCredentialsType: valueCredentialTypeUsernamePassword,
							labelCreatedBy:       valueCreatedByJX,
							labelKind:            "git",
							labelServiceKind:     "github",
						},
						Annotations: map[string]string{
							annotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							annotationURL:                    "https://github.com",
							annotationName:                   "GitHub",
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
							labelCredentialsType: valueCredentialTypeUsernamePassword,
							labelCreatedBy:       valueCreatedByJX,
							labelKind:            "git",
							labelServiceKind:     "gitlab",
						},
						Annotations: map[string]string{
							annotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://gitlab.com"),
							annotationURL:                    "https://gitlab.com",
							annotationName:                   "GitLab",
						},
					},
					Data: map[string][]byte{
						"username": []byte("test2"),
						"password": []byte("test2"),
					},
				},
			},
		},
		"save invalid config into kubernetes secret with empty credentials": {
			namespace:  "test",
			serverKind: "git",
			config: &Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test1",
							},
						},
						Name:        "GitHub",
						Kind:        "github",
						CurrentUser: "test1",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: true,
		},
		"save invalid config into kubernetes secret with empty username": {
			namespace:  "test",
			serverKind: "git",
			config: &Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								ApiToken: "test1",
							},
						},
						Name:        "GitHub",
						Kind:        "github",
						CurrentUser: "test1",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: true,
		},
		"update config into kubernetes secret": {
			namespace:  "test",
			serverKind: "git",
			setup: func(t *testing.T, client kubernetes.Interface, ns string) {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							labelCredentialsType: valueCredentialTypeUsernamePassword,
							labelCreatedBy:       valueCreatedByJX,
							labelKind:            "git",
							labelServiceKind:     "github",
						},
						Annotations: map[string]string{
							annotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							annotationURL:                    "https://github.com",
							annotationName:                   "GitHub",
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
			config: &Config{
				Servers: []Server{
					{
						URL: "https://github.com",
						Users: []User{
							{
								Username: "test1",
								ApiToken: "test1",
							},
							{
								Username: "test2",
								ApiToken: "test2",
							},
						},
						Name:        "GitHub",
						Kind:        "github",
						CurrentUser: "test1",
					},
				},
				CurrentServer: "https://github.com",
			},
			err: false,
			want: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "jx-pipeline-git-github-github",
						Namespace: "test",
						Labels: map[string]string{
							labelCredentialsType: valueCredentialTypeUsernamePassword,
							labelCreatedBy:       valueCreatedByJX,
							labelKind:            "git",
							labelServiceKind:     "github",
						},
						Annotations: map[string]string{
							annotationCredentialsDescription: fmt.Sprintf("Configuration and credentials for server https://github.com"),
							annotationURL:                    "https://github.com",
							annotationName:                   "GitHub",
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
			svc := NewKubeConfigService(client, tc.namespace, tc.serverKind, "")
			svc.SetConfig(tc.config)
			err := svc.SaveConfig()
			if tc.err {
				assert.Error(t, err, "should save config into secret with an error")
			} else {
				assert.NoError(t, err, "should save config into secret without an error")
				for _, wantSecret := range tc.want {
					gotSecret, err := client.CoreV1().Secrets(tc.namespace).Get(wantSecret.Name, metav1.GetOptions{})
					assert.NoErrorf(t, err, "should find secret %q", wantSecret.Name)
					if gotSecret == nil {
						t.Fatalf("created secret %q should not be nil", wantSecret.Name)
					}
					assert.Equal(t, *wantSecret, *gotSecret)
				}
			}
		})
	}
}
