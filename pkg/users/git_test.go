package users_test

import (
	"fmt"
	"testing"

	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/jenkins-x/jx/pkg/users"

	uuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/gits"
	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFindUserByLabel(t *testing.T) {
	t.Parallel()
	resolver, _, err := prepare(t)
	assert.NoError(t, err)
	gitUserID := uuid.NewV4().String()
	// Create the user
	labeledUserID, err := createDummyUser(resolver, true, gitUserID)
	defer func() {
		err := removeDummyUser(labeledUserID, resolver)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	unLabeledUserID, err := createDummyUser(resolver, false, gitUserID)
	defer func() {
		err := removeDummyUser(unLabeledUserID, resolver)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	gitUser := gits.GitUser{
		Login: gitUserID,
	}
	user, err := resolver.Resolve(&gitUser)
	assert.NoError(t, err)
	// Validate that we found the labeled one, not the unlabeled one
	assert.Equal(t, labeledUserID, user.Spec.Login)
}

func TestFindUserBySignature(t *testing.T) {
	t.Parallel()
	resolver, _, err := prepare(t)
	assert.NoError(t, err)
	gitUserID := uuid.NewV4().String()
	// Create the user
	userID, err := createDummyUser(resolver, true, gitUserID)
	defer func() {
		err := removeDummyUser(userID, resolver)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	assert.NoError(t, err)
	signature := object.Signature{
		Email: fmt.Sprintf("%s@test.com", userID),
		Name:  userID,
	}
	user, err := resolver.GitSignatureAsUser(&signature)
	assert.NoError(t, err)
	// Validate that we found the labeled one, not the unlabeled one
	assert.Equal(t, userID, user.Spec.Login)
}

func TestFindUserByAccountReference(t *testing.T) {
	t.Parallel()
	resolver, _, err := prepare(t)
	assert.NoError(t, err)
	gitUserID1 := uuid.NewV4().String()
	gitUserID2 := uuid.NewV4().String()
	// Create the user
	userID1, err := createDummyUser(resolver, false, gitUserID1)
	defer func() {
		err := removeDummyUser(userID1, resolver)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	userID2, err := createDummyUser(resolver, false, gitUserID2)
	defer func() {
		err := removeDummyUser(userID2, resolver)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	gitUser := gits.GitUser{
		Login: gitUserID1,
	}
	user, err := resolver.Resolve(&gitUser)
	assert.NoError(t, err)
	// Validate that we found the correct user
	assert.Equal(t, userID1, user.Spec.Login)
	// Validate that the label was added
	val, ok := user.Labels[resolver.GitProviderKey()]
	assert.True(t, ok)
	assert.Equal(t, gitUserID1, val)
}

func TestFindUserByFromGitProvider(t *testing.T) {
	t.Parallel()
	resolver, fakeProvider, err := prepare(t)
	assert.NoError(t, err)
	gitUserID1 := uuid.NewV4().String()
	gitUserID2 := uuid.NewV4().String()
	// Create the user
	gitUser1 := &gits.GitUser{
		Name:  uuid.NewV4().String(),
		Email: fmt.Sprintf("%s@test.com", uuid.NewV4().String()),
		Login: gitUserID1,
	}
	fakeProvider.Users = []*gits.GitUser{
		gitUser1,
	}
	userID2, err := createDummyUser(resolver, false, gitUserID2)
	defer func() {
		err := removeDummyUser(userID2, resolver)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)
	gitUser := gits.GitUser{
		Login: gitUserID1,
	}
	user, err := resolver.Resolve(&gitUser)
	assert.NoError(t, err)
	// Validate that we create a new user from the git provider
	assert.Equal(t, gitUser1.Email, user.Spec.Email)
	assert.Equal(t, gitUser1.Name, user.Spec.Name)
	// Validate that the label was added
	val, ok := user.Labels[resolver.GitProviderKey()]
	assert.True(t, ok)
	assert.Equal(t, gitUserID1, val)
	// Validate that the account reference is created
	assert.Len(t, user.Spec.Accounts, 1)
	assert.Equal(t, user.Spec.Accounts[0].Provider, resolver.GitProviderKey())
	assert.Equal(t, user.Spec.Accounts[0].ID, gitUserID1)
}

func prepare(t *testing.T) (*users.GitUserResolver, *gits.FakeProvider, error) {
	testOrgName := "myorg"
	testRepoName := "my-app"
	fakeRepo := gits.NewFakeRepository(testOrgName, testRepoName)
	fakeProvider := gits.NewFakeProvider(fakeRepo)
	fakeProvider.Type = gits.Fake

	o := cmd.CommonOptions{}
	cmd.ConfigureTestOptionsWithResources(&o,
		[]runtime.Object{},
		[]runtime.Object{},
		&gits.GitFake{},
		fakeProvider,
		helm_test.NewMockHelmer(),
		resources_test.NewMockInstaller(),
	)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return nil, nil, err
	}

	return &users.GitUserResolver{
		GitProvider: fakeProvider,
		JXClient:    jxClient,
		Namespace:   ns,
	}, fakeProvider, nil
}

func createDummyUser(resolver *users.GitUserResolver, createLabels bool, gitUserID string) (string, error) {
	id := uuid.NewV4().String()
	spec := jenkinsv1.UserDetails{
		Name:  id,
		Email: fmt.Sprintf("%s@test.com", id),
		Login: id,
		Accounts: []jenkinsv1.AccountReference{
			jenkinsv1.AccountReference{
				ID:       gitUserID,
				Provider: resolver.GitProviderKey(),
			},
		},
	}
	meta := metav1.ObjectMeta{
		Name: id,
	}
	if createLabels {
		meta.Labels = map[string]string{
			resolver.GitProviderKey(): gitUserID,
		}
	}
	_, err := resolver.JXClient.JenkinsV1().Users(resolver.Namespace).Create(&jenkinsv1.User{
		Spec:       spec,
		ObjectMeta: meta,
	})
	return id, err
}

func removeDummyUser(id string, resolver *users.GitUserResolver) error {
	return resolver.JXClient.JenkinsV1().Users(resolver.Namespace).Delete(id, &metav1.DeleteOptions{})
}
