package local

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
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

func TestGetTargetContainerPort(t *testing.T) {
	containersTest1 := []v1.Container{
		{Name: "anothercontainer", Ports: []v1.ContainerPort{{ContainerPort: 3000}}},
		{Name: "mycontainer", Ports: []v1.ContainerPort{{ContainerPort: 4000}}},
		{Name: "multi-port", Ports: []v1.ContainerPort{{ContainerPort: 80}, {ContainerPort: 81}}},
		{Name: "no-port", Ports: []v1.ContainerPort{{}}},
	}

	testCases := []struct {
		description     string
		containers      []v1.Container
		targetContainer string
		expectedPorts   []int
		expectErr       bool
	}{
		{"test correct container and port found", containersTest1, "mycontainer", []int{4000}, false},
		{"test container not found error", containersTest1, "randomcontainer", []int{0}, true},
		{"container found, multiple ports", containersTest1, "multi-port", []int{80, 81}, false},
		{"container found, no ports", containersTest1, "no-port", []int{0}, false},
	}

	for _, tc := range testCases {
		ports, err := getTargetContainerPorts(tc.containers, tc.targetContainer)
		if tc.expectErr && err == nil {
			t.Errorf("Expected err but did not get one for case: %s", tc.description)
		}

		if (!areEqual(tc.expectedPorts, []int{0})) && (!areEqual(tc.expectedPorts, ports)) {
			t.Errorf("Expected port %v, got %v for scenario: %v", tc.expectedPorts, ports, tc.description)
		}
	}
}

func TestGetPortMapping(t *testing.T) {
	testCases := []struct {
		description     string
		portMapping     []string
		expectedMapping map[int]int
		expectedErr     bool
	}{
		{"single port mapping", []string{"8080:8080"}, map[int]int{8080: 8080}, false},
		{"multiple port mappings", []string{"8080:80", "8081:81"}, map[int]int{80: 8080, 81: 8081}, false},
		{"multiple port mappings, same local port", []string{"8080:8080", "8080:8081"}, map[int]int{}, true},
		{"multiple port mappings, same remote port", []string{"8081:8081", "8080:8081"}, map[int]int{}, true},
	}

	for _, tc := range testCases {
		m, err := getPortMapping(tc.portMapping)

		if tc.expectedErr {
			if err == nil {
				t.Errorf("Expected err but did not get one for test case: %s", tc.description)
			}
		} else {
			if !reflect.DeepEqual(m, tc.expectedMapping) {
				t.Errorf("For scenario %v - expected: %v, actual: %v", tc.description, tc.expectedMapping, m)
			}
		}
	}
}

func areEqual(a, b []int) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
