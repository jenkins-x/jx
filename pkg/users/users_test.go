package users_test

import (
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/jenkins-x/jx/pkg/users"

	helm_test "github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"k8s.io/apimachinery/pkg/runtime"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/stretchr/testify/assert"
)

func TestResolveUserWithEmptyIdDoesNotCreateEmptyAccountReference(t *testing.T) {
	t.Parallel()
	o := cmd.CommonOptions{}
	user := jenkinsv1.User{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "jx",
			Name:      "test-user",
		},
		Spec: jenkinsv1.UserDetails{
			Name:     "Test",
			Email:    "test@test.com",
			Accounts: make([]jenkinsv1.AccountReference, 0),
		},
	}

	cmd.ConfigureTestOptionsWithResources(&o,
		[]runtime.Object{},
		[]runtime.Object{&user},
		&gits.GitFake{},
		&gits.FakeProvider{},
		helm_test.NewMockHelmer(),
		resources_test.NewMockInstaller(),
	)

	jxClient, ns, err := o.JXClientAndDevNamespace()
	assert.NoError(t, err)

	// First check resolving with an id
	resolved, err := users.Resolve("", "testProvider", jxClient, ns, func(id string, users []jenkinsv1.User) (string,
		[]jenkinsv1.User, *jenkinsv1.User, error) {
		return "123", []jenkinsv1.User{user}, &jenkinsv1.User{}, nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, resolved)
	assert.Len(t, resolved.Spec.Accounts, 1)

	// Now with an empty id
	resolved, err = users.Resolve("", "testProvider", jxClient, ns, func(id string, users []jenkinsv1.User) (string,
		[]jenkinsv1.User, *jenkinsv1.User, error) {
		return "", []jenkinsv1.User{user}, &jenkinsv1.User{}, nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, resolved)
	assert.Len(t, resolved.Spec.Accounts, 0)
}
