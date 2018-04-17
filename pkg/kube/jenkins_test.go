package kube

import (
	"testing"

	"github.com/beevik/etree"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAddGiteaServers(t *testing.T) {
	cm := &corev1.ConfigMap{
		Data: map[string]string{},
	}

	expectedGitUrl := "https://github.bees.com"
	expectedGitName := "mygit"
	expectedCredentials := "my-credential-name"
	server := &auth.AuthServer{
		Kind: gits.KindGitea,
		Name: expectedGitName,
		URL:  expectedGitUrl,
	}
	userAuth := &auth.UserAuth{
		Username: "dummy",
	}

	updated, err := UpdateJenkinsGitServers(cm, server, userAuth, expectedCredentials)
	assert.Nil(t, err, "Failed to update the ConfigMap for server %s", expectedGitUrl)
	assert.True(t, updated, "Should have updated the ConfigMap for server %s", expectedGitUrl)

	for k, v := range cm.Data {
		tests.Debugf("Updated the ConfigMap: %s = %s\n", k, v)
	}

	doc, _, err := parseXml(cm.Data[giteaConfigMapKey])
	assert.Nil(t, err, "Failed to parse resulting xml for server %s", expectedGitUrl)
	assertElementValues(t, doc, "//serverUrl", expectedGitUrl)

	updated, err = UpdateJenkinsGitServers(cm, server, userAuth, expectedCredentials)
	assert.Nil(t, err, "Failed to update the ConfigMap for server %s", expectedGitUrl)
	assert.False(t, updated, "Should not have updated the ConfigMap for server %s", expectedGitUrl)

	doc, _, err = parseXml(cm.Data[giteaConfigMapKey])
	assert.Nil(t, err, "Failed to parse resulting xml for server %s", expectedGitUrl)
	assertElementValues(t, doc, "//serverUrl", expectedGitUrl)

	// lets add an extra service to an existing one
	cm.Data[giteaConfigMapKey] = `<?xml version='1.1' encoding='UTF-8'?>
<org.jenkinsci.plugin.gitea.servers.GiteaServers plugin="gitea@1.0.5">
  <servers>
    <org.jenkinsci.plugin.gitea.servers.GiteaServer>
      <displayName>gitea</displayName>
      <serverUrl>http://gitea.changeme.com</serverUrl>
      <manageHooks>true</manageHooks>
      <credentialsId>jenkins-x-gitea</credentialsId>
    </org.jenkinsci.plugin.gitea.servers.GiteaServer>
  </servers>
</org.jenkinsci.plugin.gitea.servers.GiteaServers>
`
	updated, err = UpdateJenkinsGitServers(cm, server, userAuth, expectedCredentials)
	assert.Nil(t, err, "Failed to update the ConfigMap for server %s", expectedGitUrl)
	assert.True(t, updated, "Should have updated the ConfigMap for server %s", expectedGitUrl)

	doc, _, err = parseXml(cm.Data[giteaConfigMapKey])
	assert.Nil(t, err, "Failed to parse resulting xml for server %s", expectedGitUrl)
	assertElementValues(t, doc, "//serverUrl", "http://gitea.changeme.com", expectedGitUrl)

	// lets modify an existing credentials value
	// lets add an extra service to an existing one
	cm.Data[giteaConfigMapKey] = `<?xml version='1.1' encoding='UTF-8'?>
<org.jenkinsci.plugin.gitea.servers.GiteaServers plugin="gitea@1.0.5">
  <servers>
    <org.jenkinsci.plugin.gitea.servers.GiteaServer>
      <displayName>gitea</displayName>
      <serverUrl>` + expectedGitUrl + `</serverUrl>
      <manageHooks>true</manageHooks>
      <credentialsId>jenkins-x-gitea</credentialsId>
    </org.jenkinsci.plugin.gitea.servers.GiteaServer>
  </servers>
</org.jenkinsci.plugin.gitea.servers.GiteaServers>
`

	updated, err = UpdateJenkinsGitServers(cm, server, userAuth, expectedCredentials)
	assert.Nil(t, err, "Failed to update the ConfigMap for server %s", expectedGitUrl)
	assert.True(t, updated, "Should have updated the ConfigMap for server %s", expectedGitUrl)

	doc, _, err = parseXml(cm.Data[giteaConfigMapKey])
	assert.Nil(t, err, "Failed to parse resulting xml for server %s", expectedGitUrl)
	assertElementValues(t, doc, "//serverUrl", expectedGitUrl)
	assertElementValues(t, doc, "//credentialsId", expectedCredentials)
}

func assertElementValues(t *testing.T, doc *etree.Document, path string, expectedValues ...string) {
	elements := doc.FindElements(path)
	actuals := []string{}
	for _, e := range elements {
		actuals = append(actuals, e.Text())
	}
	assert.EqualValues(t, expectedValues, actuals, "Invalid values for path %s", path)
}
