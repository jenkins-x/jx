package builder

import (
	"encoding/base64"
	"encoding/json"

	"github.com/docker/docker/api/types"
)

// DockerConfigEntryWithAuth is used solely for translating docker's AuthConfig token
// into a credentialprovider.dockerConfigEntry during JSON deserialization.
//
// pulled from https://github.com/kubernetes/kubernetes/blob/97892854cfa736315378cc2c206a7f4b3e190d05/pkg/credentialprovider/config.go#L232-L243
type DockerConfigEntryWithAuth struct {
	// +optional
	Username string `json:"username,omitempty"`
	// +optional
	Password string `json:"password,omitempty"`
	// +optional
	Email string `json:"email,omitempty"`
	// +optional
	Auth string `json:"auth,omitempty"`
}

// FromAuthConfigToken converts a docker auth token into type DockerConfigEntryWithAuth. This allows us to
// Marshal the object into a Kubernetes registry auth secret.
func FromAuthConfigToken(authToken string) (*DockerConfigEntryWithAuth, error) {
	data, err := base64.StdEncoding.DecodeString(authToken)
	if err != nil {
		return nil, err
	}
	var regAuth types.AuthConfig
	if err := json.Unmarshal(data, &regAuth); err != nil {
		return nil, err
	}
	return FromAuthConfig(regAuth), nil
}

// FromAuthConfig converts a docker auth token into type DockerConfigEntryWithAuth. This allows us to
// Marshal the object into a Kubernetes registry auth secret.
func FromAuthConfig(ac types.AuthConfig) *DockerConfigEntryWithAuth {
	return &DockerConfigEntryWithAuth{
		Username: ac.Username,
		Password: ac.Password,
		Email:    ac.Email,
		Auth:     ac.Auth,
	}
}
