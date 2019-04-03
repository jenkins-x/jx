package v1

import (
	"encoding/json"
	"errors"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"testing"
)

var (
	testApp = &v1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1.AppSpec{},
	}
)

func TestPatchUpdateAppNoModification(t *testing.T) {
	appJSON, err := json.Marshal(testApp)
	if err != nil {
		assert.Failf(t, "unable to marshal test instance: %s", err.Error())
	}
	get := func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: defaultHeaders(), Body: bytesBody(appJSON)}, nil
	}

	patch := func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: defaultHeaders(), Body: bytesBody(appJSON)}, nil
	}

	fakeClient := newClientForTest(get, patch)

	apps := apps{
		client: fakeClient,
		ns:     "default",
	}

	updatedApp, err := apps.PatchUpdate(testApp)
	assert.NoError(t, err)
	assert.Equal(t, testApp, updatedApp)
}

func TestPatchUpdateAppWithChange(t *testing.T) {
	services := []string{"foo", "bar"}
	clonedApp := testApp.DeepCopy()
	clonedApp.Spec.ExposedServices = services

	get := func(*http.Request) (*http.Response, error) {
		appJSON, err := json.Marshal(testApp)
		if err != nil {
			assert.Failf(t, "unable to marshal test instance: %s", err.Error())
		}
		return &http.Response{StatusCode: 200, Header: defaultHeaders(), Body: bytesBody(appJSON)}, nil
	}

	patch := func(*http.Request) (*http.Response, error) {
		appJSON, err := json.Marshal(clonedApp)
		if err != nil {
			assert.Failf(t, "unable to marshal test instance: %s", err.Error())
		}
		return &http.Response{StatusCode: 200, Header: defaultHeaders(), Body: bytesBody(appJSON)}, nil
	}

	fakeClient := newClientForTest(get, patch)

	apps := apps{
		client: fakeClient,
		ns:     "default",
	}

	updatedApp, err := apps.PatchUpdate(testApp)
	assert.NoError(t, err)
	assert.NotEqual(t, testApp, updatedApp)
	assert.Equal(t, services, updatedApp.Spec.ExposedServices)
}

func TestPatchUpdateAppWithErrorInGet(t *testing.T) {
	errorMessage := "error during GET"
	get := func(*http.Request) (*http.Response, error) {
		return nil, errors.New(errorMessage)
	}

	fakeClient := newClientForTest(get, nil)

	apps := apps{
		client: fakeClient,
		ns:     "default",
	}

	updatedApp, err := apps.PatchUpdate(testApp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errorMessage)
	assert.Nil(t, updatedApp)
}

func TestPatchUpdateAppWithErrorInPatch(t *testing.T) {
	errorMessage := "error during PATCH"
	get := func(*http.Request) (*http.Response, error) {
		appJSON, err := json.Marshal(testApp)
		if err != nil {
			assert.Failf(t, "unable to marshal test instance: %s", err.Error())
		}
		return &http.Response{StatusCode: 200, Header: defaultHeaders(), Body: bytesBody(appJSON)}, nil
	}

	patch := func(*http.Request) (*http.Response, error) {
		return nil, errors.New(errorMessage)
	}

	fakeClient := newClientForTest(get, patch)

	apps := apps{
		client: fakeClient,
		ns:     "default",
	}

	updatedApp, err := apps.PatchUpdate(testApp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errorMessage)
	assert.Nil(t, updatedApp)
}
