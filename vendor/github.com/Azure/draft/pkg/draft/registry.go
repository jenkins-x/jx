package draft

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const (
	// name of the docker pull secret draftd will create in the desired destination namespace
	pullSecretName = "draftd-pullsecret"
	// name of the default service account draftd will modify with the imagepullsecret
	svcAcctNameDefault = "default"
)

type (
	// RegistryConfig specifies configuration for the image repository.
	RegistryConfig struct {
		// Auth is the authorization token used to push images up to the registry.
		Auth string
		// URL is the URL of the registry (e.g. quay.io/myuser, docker.io/myuser, myregistry.azurecr.io)
		URL string
	}

	// RegistryAuth is the registry authentication credentials
	RegistryAuth struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		Email         string `json:"email"`
		RegistryToken string `json:"registrytoken"`
	}

	// DockerAuth is a container for the registry authentication credentials wrapped
	// by the registry server name.
	DockerAuth map[string]RegistryAuth
)

func configureRegistryAuth(auth string) (RegistryAuth, error) {
	var regauth RegistryAuth
	// base64 decode the registryauth string.
	b64dec, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return regauth, fmt.Errorf("could not base64 decode registry authentication string: %v", err)
	}
	// break up registry auth json string into a RegistryAuth object.
	if err := json.Unmarshal(b64dec, &regauth); err != nil {
		return regauth, fmt.Errorf("could not json decode registry authentication string: %v", err)
	}

	return regauth, nil
}
