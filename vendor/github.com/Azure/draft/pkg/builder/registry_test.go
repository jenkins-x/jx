package builder

import (
	"reflect"
	"testing"
)

func TestFromAuthConfigToken(t *testing.T) {
	var authConfigTests = []struct {
		input    string
		fail     bool
		expected *DockerConfigEntryWithAuth
	}{
		{"", true, nil},
		{"badbase64input", true, nil},
		{"e30K", false, &DockerConfigEntryWithAuth{}},
		{"eyJ1c2VybmFtZSI6InVzZXJuYW1lIiwicGFzc3dvcmQiOiJwYXNzd29yZCJ9Cg==", false, &DockerConfigEntryWithAuth{Username: "username", Password: "password"}},
		{"eyJ1c2VybmFtZSI6InVzZXJuYW1lIiwicGFzc3dvcmQiOiJwYXNzd29yZCIsImVtYWlsIjoiZW1haWwiLCJhdXRoIjoiYXV0aCJ9Cg==", false, &DockerConfigEntryWithAuth{Username: "username", Password: "password", Email: "email", Auth: "auth"}},
		{"eyJ1c2VybmFtZSI6InVzZXJuYW1lIiwicGFzc3dvcmQiOiJwYXNzd29yZCIsImVtYWlsIjoiZW1haWwiLCJhdXRoIjoiYXV0aCIsInNlcnZlcmFkZHJlc3MiOiJodHRwOi8vc2VydmVyYWRkcmVzcy5jb20ifQo=", false, &DockerConfigEntryWithAuth{Username: "username", Password: "password", Email: "email", Auth: "auth"}},
	}

	for _, tt := range authConfigTests {
		actual, err := FromAuthConfigToken(tt.input)
		if tt.fail && err == nil {
			t.Errorf("FromAuthConfigToken(%s) was expected to fail", tt.input)
		} else if !tt.fail && err != nil {
			t.Errorf("FromAuthConfigToken(%s) was not expected to fail", tt.input)
		}
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("FromAuthConfigToken(%s): expected output differs from actual", tt.input)
		}
	}
}
