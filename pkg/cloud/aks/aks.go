package aks

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
)

// AzureRunner an Azure CLI runner to interact with Azure
type AzureRunner struct {
	Runner util.Commander
}

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

type containerExists struct {
	Exists bool `json:"exists"`
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

var (
	azureContainerURIRegExp = regexp.MustCompile(`https://(?P<first>\w+)\.blob\.core\.windows\.net/(?P<second>\w+)`)
)

// NewAzureRunnerWithCommander specific the command runner for Azure CLI.
func NewAzureRunnerWithCommander(runner util.Commander) *AzureRunner {
	return &AzureRunner{
		Runner: runner,
	}
}

// NewAzureRunner return a new AzureRunner
func NewAzureRunner() *AzureRunner {
	runner := &util.Command{}
	return NewAzureRunnerWithCommander(runner)
}

// GetClusterClient return AKS resource group, name and client ID.
func (az *AzureRunner) GetClusterClient(server string) (string, string, string, error) {
	clientID := ""
	group := ""
	name := ""

	clusterstr, err := az.azureCLI("aks", "list", "--query", "[].{uri:fqdn,id:servicePrincipalProfile.clientId,group:resourceGroup,name:name}")
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
func (az *AzureRunner) GetRegistry(azureRegistrySubscription string, resourceGroup string, name string, registry string) (string, string, string, error) {
	registryID := ""
	loginServer := registry
	dockerConfig := ""

	if registry == "" {
		loginServer = name + ".azurecr.io"
	}

	if !strings.HasSuffix(loginServer, "azurecr.io") {
		return dockerConfig, loginServer, registryID, nil
	}

	acrRG, acrName, registryID, err := az.getRegistryID(azureRegistrySubscription, loginServer)
	if err != nil {
		return dockerConfig, loginServer, registryID, err
	}
	// not exist and create a new one in resourceGroup
	if registryID == "" {
		acrRG = resourceGroup
		acrName = name
		registryID, loginServer, err = az.createRegistry(azureRegistrySubscription, acrRG, acrName)
		if err != nil {
			return dockerConfig, loginServer, registryID, err
		}
	}
	dockerConfig, err = az.getACRCredential(azureRegistrySubscription, acrRG, acrName)
	return dockerConfig, loginServer, registryID, err
}

// AssignRole Assign the client a reader role for registry.
func (az *AzureRunner) AssignRole(client string, registry string) {
	if client == "" || registry == "" {
		return
	}
	az.azureCLI("role", "assignment", "create", "--assignee", client, "--role", "Reader", "--scope", registry) //nolint:errcheck
}

// getRegistryID returns acrRG, acrName, acrID, error
func (az *AzureRunner) getRegistryID(azureRegistrySubscription string, loginServer string) (string, string, string, error) {
	acrRG := ""
	acrName := ""
	acrID := ""

	acrListArgs := []string{
		"acr",
		"list",
		"--query",
		"[].{uri:loginServer,id:id,name:name,group:resourceGroup}",
	}

	if azureRegistrySubscription != "" {
		acrListArgs = append(acrListArgs, "--subscription", azureRegistrySubscription)
	}

	acrList, err := az.azureCLI(acrListArgs...)

	if err != nil {
		log.Logger().Infof("Registry %s is not exist", util.ColorInfo(loginServer))
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
func (az *AzureRunner) createRegistry(azureRegistrySubscription string, resourceGroup string, name string) (string, string, error) {
	acrCreateArgs := []string{
		"acr",
		"create",
		"-g",
		resourceGroup,
		"-n",
		name,
		"--sku",
		"Standard",
		"--admin-enabled",
		"--query",
		"id",
		"-o",
		"tsv",
	}

	if azureRegistrySubscription != "" {
		acrCreateArgs = append(acrCreateArgs, "--subscription", azureRegistrySubscription)
	}

	registryID, err := az.azureCLI(acrCreateArgs...)
	if err != nil {
		log.Logger().Infof("Failed to create ACR %s in resource group %s", util.ColorInfo(name), util.ColorInfo(resourceGroup))
		return "", "", err
	}
	return registryID, formatLoginServer(name), nil
}

// getACRCredential return .dockerconfig value for the ACR
func (az *AzureRunner) getACRCredential(azureRegistrySubscription string, resourceGroup string, name string) (string, error) {
	showCredArgs := []string{
		"acr",
		"credential",
		"show",
		"-g",
		resourceGroup,
		"-n",
		name,
	}

	if azureRegistrySubscription != "" {
		showCredArgs = append(showCredArgs, "--subscription", azureRegistrySubscription)
	}

	credstr, err := az.azureCLI(showCredArgs...)
	if err != nil {
		log.Logger().Infof("Failed to get credential for ACR %s in resource group %s", util.ColorInfo(name), util.ColorInfo(resourceGroup))
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

func (az *AzureRunner) azureCLI(args ...string) (string, error) {
	az.Runner.SetName("az")
	az.Runner.SetArgs(args)
	return az.Runner.RunWithoutRetry()
}

func parseContainerURL(bucketURL string) (string, string, error) {
	match := azureContainerURIRegExp.FindStringSubmatch(bucketURL)
	if len(match) == 3 {
		return match[1], match[2], nil
	}
	return "", "", errors.New(fmt.Sprintf("Azure Blob Container Url %s could not be parsed to determine storage account and container name", bucketURL))
}

// ContainerExists checks if an Azure Storage Container exists
func (az *AzureRunner) ContainerExists(bucketURL string) (bool, error) {
	storageAccount, bucketName, err := parseContainerURL(bucketURL)
	if err != nil {
		return false, err
	}

	accessKey, err := az.GetStorageAccessKey(storageAccount)
	if err != nil {
		return false, err
	}

	bucketExistsArgs := []string{
		"storage",
		"container",
		"exists",
		"-n",
		bucketName,
		"--account-name",
		storageAccount,
		"--account-key",
		accessKey,
	}

	cmdResult, err := az.azureCLI(bucketExistsArgs...)

	if err != nil {
		log.Logger().Infof("Error checking bucket exists: %s, %s", cmdResult, err)
		return false, err
	}

	containerExists := containerExists{}
	err = json.Unmarshal([]byte(cmdResult), &containerExists)
	if err != nil {
		return false, errors.Wrap(err, "unmarshalling Azure container exists command")
	}
	return containerExists.Exists, nil

}

// CreateContainer creates a Blob container within Azure Storage
func (az *AzureRunner) CreateContainer(bucketURL string) error {
	storageAccount, bucketName, err := parseContainerURL(bucketURL)
	if err != nil {
		return err
	}

	accessKey, err := az.GetStorageAccessKey(storageAccount)
	if err != nil {
		return err
	}

	createContainerArgs := []string{
		"storage",
		"container",
		"create",
		"-n",
		bucketName,
		"--account-name",
		storageAccount,
		"--fail-on-exist",
		"--account-key",
		accessKey,
	}

	cmdResult, err := az.azureCLI(createContainerArgs...)

	if err != nil {
		log.Logger().Infof("Error creating bucket: %s, %s", cmdResult, err)
		return err
	}

	return nil
}

// GetStorageAccessKey retrieves access keys for an Azure storage account
func (az *AzureRunner) GetStorageAccessKey(storageAccount string) (string, error) {
	getStorageAccessKeyArgs := []string{
		"storage",
		"account",
		"keys",
		"list",
		"-n",
		storageAccount,
		"--query",
		"[?keyName=='key1'].value | [0]",
	}

	cmdResult, err := az.azureCLI(getStorageAccessKeyArgs...)

	if err != nil {
		return "", err
	}

	return cmdResult, nil
}
