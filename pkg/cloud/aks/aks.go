package aks

import (
	b64 "encoding/base64"
	"encoding/json"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
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

// GetClusterClient return AKS resource group, name and client ID.
func GetClusterClient(server string) (string, string, string, error) {
	clientID := ""
	group := ""
	name := ""

	clusterstr, err := azureCLI("aks", "list", "--query", "[].{uri:fqdn,id:servicePrincipalProfile.clientId,group:resourceGroup,name:name}")
	if err != nil {
		return group, name, clientID, err
	}

	clusters := []aks{}
	err = json.Unmarshal([]byte(clusterstr), &clusters)
	if err != nil {
		return group, name, clientID, err
	}

	for _, v := range clusters {
		if "https://"+v.URI+":443" == server {
			clientID = v.ID
			name = v.Name
			group = v.Group
			break
		}
	}

	return group, name, clientID, err
}


 // GetRegistry Return the docker registry config, registry login server and resource id, error
func GetRegistry(resourceGroup string, name string, registry string) (string, string, string, error) {
	registryID := ""
	loginServer := registry
	dockerConfig := ""

	if registry == "" {
		loginServer = name + ".azurecr.io"
	}

	if !strings.HasSuffix(loginServer, "azurecr.io") {
		return dockerConfig, loginServer, registryID, nil
	}

	acrRG, acrName, registryID, err := getRegistryID(loginServer)
	if err != nil {
		return dockerConfig, loginServer, registryID, err
	}
	// not exist and create a new one in resourceGroup
	if registryID == "" {
		acrRG = resourceGroup
		acrName = name
		registryID, loginServer, err = createRegistry(acrRG, acrName)
		if err != nil {
			return dockerConfig, loginServer, registryID, err
		}
	}
	dockerConfig, err = getACRCredential(acrRG, acrName)
	return dockerConfig, loginServer, registryID, err
}

// AssignRole Assign the client a reader role for registry.
func AssignRole(client string, registry string) {
	if client == "" || registry == "" {
		return
	}
	azureCLI("role", "assignment", "create", "--assignee", client, "--role", "Reader", "--scope", registry)
}

// getRegistryID returns acrRG, acrName, acrID, error
func getRegistryID(loginServer string) (string, string, string, error) {
	acrRG := ""
	acrName := ""
	acrID := ""

	acrList, err := azureCLI("acr", "list", "--query", "[].{uri:loginServer,id:id,name:name,group:resourceGroup}")
	if err != nil {
		log.Infof("Registry %s is not exist\n", util.ColorInfo(loginServer))
	} else {
		registries := []acr{}
		err = json.Unmarshal([]byte(acrList), &registries)
		if err != nil {
			return "", "", "", err
		}
		for _, v := range registries {
			if v.URI == loginServer {
				acrID = v.ID
				acrRG = v.Group
				acrName = v.Name
				break
			}
		}
	}
	return acrRG, acrName, acrID, nil
}

// createRegistry return resource ID, login server and error
func createRegistry(resourceGroup string, name string) (string, string, error) {
	registryID, err := azureCLI("acr", "create", "-g", resourceGroup, "-n", name, "--sku", "Standard", "--admin-enabled", "--query", "id")
	if err != nil {
		log.Infof("Failed to create ACR %s in resource group %s\n", util.ColorInfo(name), util.ColorInfo(resourceGroup))
		return "", "", err	
	}
	return string(registryID), formatLoginServer(name), nil
}

// getACRCredential return .dockerconfig value for the ACR
func getACRCredential(resourceGroup string, name string) (string, error) {
	credstr, err := azureCLI("acr", "credential", "show", "-g", resourceGroup, "-n", name)
	if err != nil {
		log.Infof("Failed to get credential for ACR %s in resource group %s\n", util.ColorInfo(name), util.ColorInfo(resourceGroup))
		return "", err	
	}
	cred := credential{}
	err = json.Unmarshal([]byte(credstr), &cred)
	if err != nil {
		return "", err	
	}
	newSecret := &auth{}
	dockerConfig := &config{}
	newSecret.Auth = b64.StdEncoding.EncodeToString([]byte(cred.Username + ":" + cred.Passwords[0].Value))
	if dockerConfig.Auths == nil {
		dockerConfig.Auths = map[string]*auth{}
	}
	dockerConfig.Auths[formatLoginServer(name)] = newSecret
	dockerConfigStr, err := json.Marshal(dockerConfig)
	return string(dockerConfigStr), err
}

func formatLoginServer(name string) string {
	return name + ".azurecr.io"
}

func azureCLI(args ...string) (string, error) {
	cmd := util.Command{
		Name: "az",
		Args: args,
	}
	return cmd.RunWithoutRetry()
}