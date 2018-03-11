package local

import (
	"reflect"
	"testing"

	"k8s.io/client-go/pkg/api/v1"
)

func TestDeployedApplication(t *testing.T) {
	expectedApp := &App{
		Name:      "example-app",
		Namespace: "example-namespace",
	}

	app, err := DeployedApplication("../testdata/app/draft.toml", "development")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedApp, app) {
		t.Errorf("Expected %#v, got %#v", expectedApp, app)
	}
}

func TestGetContainerPort(t *testing.T) {
	containersTest1 := []v1.Container{
		v1.Container{Name: "anothercontainer", Ports: []v1.ContainerPort{v1.ContainerPort{ContainerPort: 3000}}},
		v1.Container{Name: "mycontainer", Ports: []v1.ContainerPort{v1.ContainerPort{ContainerPort: 4000}}},
	}

	testCases := []struct {
		description     string
		containers      []v1.Container
		targetContainer string
		expectedPort    int
		expectErr       bool
	}{
		{"test correct container and port found", containersTest1, "mycontainer", 4000, false},
		{"test first container and port found", containersTest1, "", 3000, false},
		{"test container not found error", containersTest1, "randomcontainer", 0, true},
	}

	for _, tc := range testCases {
		port, err := getContainerPort(tc.containers, tc.targetContainer)
		if tc.expectErr && err == nil {
			t.Errorf("Expected err but did not get one for case: %s", tc.description)
		}

		if tc.expectedPort != 0 && (tc.expectedPort != port) {
			t.Errorf("Expected port %v, got %v for scenario: %s", tc.expectedPort, port, tc.description)
		}
	}
}
