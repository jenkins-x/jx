package aks

import (
	b64 "encoding/base64"
	"encoding/json"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"os/exec"
	"strings"
)

type aks struct {
	ID    string `json:"id"`
	URI   string `json:"uri"`
	Group string `json:"group"`
	Name  string `json:"name"`
}

type acr struct {
	ID    string `json:"id"`
	URI   string `json:"uri"`
	Group string `json:"group"`
	Name  string `json:"name"`
}

type password struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type credential struct {
	Passwords []password `json:"passwords"`
	Username  string     `json:"username"`
}

type auth struct {
	Auth string `json:"auth,omitempty"`
}

type config struct {
	Auths map[string]*auth `json:"auths,omitempty"`
}

func GetClusterClient(server string) (string, string, string, error) {
	clusterstr, err := exec.Command("az", "aks", "list", "--query", "[].{uri:fqdn,id:servicePrincipalProfile.clientId,group:resourceGroup,name:name}").Output()
	if err != nil {
		return "", "", "", err
	}

	clusters := []aks{}
	json.Unmarshal(clusterstr, &clusters)

	clientId := ""
	group := ""
	name := ""
	for _, v := range clusters {
		if "https://"+v.URI+":443" == server {
			clientId = v.ID
			name = v.Name
			group = v.Group
			break
		}
	}

	return group, name, clientId, nil
}


 // GetRegistery Return the docker registry config, registry uri and resource id, error
func GetRegistry(resourceGroup string, name string, registry string) (string, string, string, error) {
	registryID := ""

	if registry == "" {
		registry = name + ".azurecr.io"
	}

	if !strings.HasSuffix(registry, "azurecr.io") {
		return "", "", "", nil
	}

	registriesstr, err := exec.Command("az", "acr", "list", "--query", "[].{uri:loginServer,id:id,name:name,group:resourceGroup}").Output()
	if err != nil {
		log.Infof("Registry %s not found, create a new one %s in resource group %s\n", util.ColorInfo(registry), util.ColorInfo(name), util.ColorInfo(resourceGroup))
	} else {
		registries := []acr{}
		json.Unmarshal(registriesstr, &registries)

		for _, v := range registries {
			if v.URI == registry {
				registryID = v.ID
				resourceGroup = v.Group
				name = v.Name
				break
			}
		}
	}

	// not exist and create a new one in resourceGroup
	if registryID == "" {
		registryIDStr, err := exec.Command("az", "acr", "create", "-g", resourceGroup, "-n", name, "--sku", "Standard", "--admin-enabled", "--query", "id").Output()
		registryID = string(registryIDStr)
		if err != nil {
			log.Infof("Failed to create ACR %s in resource group %s\n", util.ColorInfo(name), util.ColorInfo(resourceGroup))
			return "", "", "", err
		}
		registry = name + ".azurecr.io"
	}

	credstr, err := exec.Command("az", "acr", "credential", "show", "-g", resourceGroup, "-n", name).Output()
	cred := credential{}
	json.Unmarshal(credstr, &cred)

	newSecret := &auth{}
	dockerConfig := &config{}
	newSecret.Auth = b64.StdEncoding.EncodeToString([]byte(cred.Username + ":" + cred.Passwords[0].Value))
	if dockerConfig.Auths == nil {
		dockerConfig.Auths = map[string]*auth{}
	}
	dockerConfig.Auths[registry] = newSecret

	dockerConfigStr, err := json.Marshal(dockerConfig)

	if err != nil {
		log.Infof("Failed to get credentials for ACR %s in resource group %s\n", util.ColorInfo(name), util.ColorInfo(resourceGroup))
		return "", "", "", err
	}

	return string(dockerConfigStr), registry, registryID, nil
}

// AssignRole Assign the client a reader role for registry.
func AssignRole(client string, registry string) {
	if client == "" || registry == "" {
		return
	}
	exec.Command("az", "role", "assignment", "create", "--assignee", client, "--role", "Reader", "--scope", registry).Output()
}
