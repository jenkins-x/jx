package kube_test

import (
	"testing"

	jenkinsio "github.com/jenkins-x/jx/pkg/apis/jenkins.io"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/naming"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetOrCreateSourceRepositories(t *testing.T) {
	tests := []struct {
		name         string
		org          string
		repo         string
		providerURL  string
		existingName string
	}{
		{
			name:        "no existing repo",
			org:         "some-org",
			repo:        "some-repo",
			providerURL: "https://github.com",
		},
		{
			name:         "existing repo, standard name",
			org:          "some-org",
			repo:         "some-repo",
			providerURL:  "https://github.com",
			existingName: "some-org-some-repo",
		},
		{
			name:         "existing repo, different name",
			org:          "some-org",
			repo:         "some-repo",
			providerURL:  "https://github.com",
			existingName: "some-arbitrary-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := "jx"

			var existingRepo *v1.SourceRepository
			if tt.existingName != "" {
				existingRepo = &v1.SourceRepository{
					TypeMeta: metav1.TypeMeta{
						Kind:       "SourceRepository",
						APIVersion: jenkinsio.GroupName + "/" + jenkinsio.Version,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      tt.existingName,
						Namespace: ns,
						Labels: map[string]string{
							v1.LabelOwner:      tt.org,
							v1.LabelRepository: tt.repo,
							v1.LabelProvider:   kube.ToProviderName(tt.providerURL),
						},
					},
					Spec: v1.SourceRepositorySpec{
						Org:          tt.org,
						Provider:     tt.providerURL,
						ProviderName: kube.ToProviderName(tt.providerURL),
						Repo:         tt.repo,
					},
				}
			}
			var jxClient *fake.Clientset
			if existingRepo != nil {
				jxClient = fake.NewSimpleClientset(existingRepo)
			} else {
				jxClient = fake.NewSimpleClientset()
			}

			createdOrExisting, err := kube.GetOrCreateSourceRepository(jxClient, ns, tt.repo, tt.org, tt.providerURL)
			assert.NoError(t, err)
			assert.NotNil(t, createdOrExisting)

			if existingRepo == nil {
				createdRepoName := naming.ToValidName(tt.org + "-" + tt.repo)
				assert.Equal(t, createdRepoName, createdOrExisting.Name, "new SourceRepository name should be %s but is %s", createdRepoName, createdOrExisting.Name)
			} else {
				assert.Equal(t, existingRepo.Name, createdOrExisting.Name, "existing or updated repository name should be %s but is %s", existingRepo.Name, createdOrExisting.Name)
			}

			assert.Equal(t, tt.org, createdOrExisting.Spec.Org)
			assert.Equal(t, tt.repo, createdOrExisting.Spec.Repo)
			assert.Equal(t, kube.ToProviderName(tt.providerURL), createdOrExisting.Spec.ProviderName)
		})
	}
}
