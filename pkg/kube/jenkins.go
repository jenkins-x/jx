package kube

import (
	"strings"

	"github.com/beevik/etree"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/gits"
	corev1 "k8s.io/api/core/v1"
)

const (
	BitbucketConfigMapKey = "com.cloudbees.jenkins.plugins.bitbucket.endpoints.BitbucketEndpointConfiguration.xml"
	GiteaConfigMapKey     = "org.jenkinsci.plugin.gitea.servers.GiteaServers.xml"
	GithubConfigMapKey    = "org.jenkinsci.plugins.github_branch_source.GitHubConfiguration.xml"

	bitbucketCloudElementName  = "com.cloudbees.jenkins.plugins.bitbucket.endpoints.BitbucketCloudEndpoint"
	bitbucketServerElementName = "com.cloudbees.jenkins.plugins.bitbucket.endpoints.BitbucketServerEndpoint"

	defaultBitbucketXml = `<?xml version='1.1' encoding='UTF-8'?>
    <com.cloudbees.jenkins.plugins.bitbucket.endpoints.BitbucketEndpointConfiguration plugin="cloudbees-bitbucket-branch-source@2.2.10"/>`
)

// UpdateJenkinsGitServers update the Jenkins ConfigMap with any missing Git server configurations for the given server and token
func UpdateJenkinsGitServers(cm *corev1.ConfigMap, server *auth.ServerAuth, userAuth *auth.UserAuth, credentials string) (bool, error) {
	if gits.IsGitHubServerURL(server.URL) {
		return false, nil
	}
	var key, v1, v2 string
	var err error

	switch server.Kind {
	case gits.KindBitBucketCloud:
		key = BitbucketConfigMapKey
		v1 = cm.Data[key]
		v2, err = createBitbucketCloudConfig(v1, server, userAuth, credentials)
	case gits.KindBitBucketServer:
		key = BitbucketConfigMapKey
		v1 = cm.Data[key]
		v2, err = createBitbucketServerConfig(v1, server, userAuth, credentials)
	case gits.KindGitHub:
		key = GithubConfigMapKey
		v1 = cm.Data[key]
		v2, err = createGitHubConfig(v1, server, userAuth, credentials)
	case gits.KindGitea:
		key = GiteaConfigMapKey
		v1 = cm.Data[key]
		v2, err = createGiteaConfig(v1, server, userAuth, credentials)
	}
	if err != nil {
		return false, err
	}
	if v1 == v2 {
		return false, nil
	}
	cm.Data[key] = v2
	return true, nil
}

// ParseXml parses XML
func ParseXml(xml string) (*etree.Document, string, error) {
	prefix := ""
	doc := etree.NewDocument()
	parseXml := xml
	idx := strings.Index(xml, "?>")
	if idx > 0 {
		prefix = xml[0:idx+2] + "\n"
		parseXml = strings.TrimSpace(xml[idx+2:])
	}
	err := doc.ReadFromString(parseXml)
	return doc, prefix, err
}

func createGitHubConfig(xml string, server *auth.ServerAuth, userAuth *auth.UserAuth, credentials string) (string, error) {
	u := gits.GitHubEnterpriseApiEndpointURL(server.URL)
	if strings.TrimSpace(xml) == "" {
		xml = `<?xml version='1.1' encoding='UTF-8'?>
		    <org.jenkinsci.plugins.github__branch__source.GitHubConfiguration plugin="github-branch-source@2.3.2"/>`
	}
	answer := xml
	doc, prefix, err := ParseXml(xml)
	if err != nil {
		return answer, err
	}
	root := doc.Root()
	servers := getChild(root, "endpoints")
	if servers == nil {
		servers = root.CreateElement("endpoints")
		root.AddChild(servers)
	}

	found := false
	updated := false
	for _, n := range servers.ChildElements() {
		if getChildText(n, "apiUri") == u {
			found = true
			break
		}
	}
	if !found {
		// lets add a new element
		s := servers.CreateElement("org.jenkinsci.plugins.github__branch__source.Endpoint")
		servers.AddChild(s)
		setChildText(s, "name", server.Name)
		setChildText(s, "apiUri", u)
		updated = true
	}
	if updated {
		doc.Indent(2)
		xml2, err := doc.WriteToString()
		return prefix + xml2, err
	}
	return xml, nil

}

func createGiteaConfig(xml string, server *auth.ServerAuth, userAuth *auth.UserAuth, credentials string) (string, error) {
	u := server.URL
	if strings.TrimSpace(xml) == "" {
		xml = `<?xml version='1.1' encoding='UTF-8'?>
		    <org.jenkinsci.plugin.gitea.servers.GiteaServers plugin="gitea@1.0.5"/>`
	}
	answer := xml
	doc, prefix, err := ParseXml(xml)
	if err != nil {
		return answer, err
	}
	root := doc.Root()
	servers := getChild(root, "servers")
	if servers == nil {
		servers = root.CreateElement("servers")
		root.AddChild(servers)
	}

	found := false
	updated := false
	for _, n := range servers.ChildElements() {
		if getChildText(n, "serverUrl") == u {
			if setChildText(n, "credentialsId", credentials) {
				updated = true
			}
			found = true
			break
		}
	}
	if !found {
		// lets add a new element
		s := servers.CreateElement("org.jenkinsci.plugin.gitea.servers.GiteaServer")
		servers.AddChild(s)
		setChildText(s, "displayName", server.Name)
		setChildText(s, "serverUrl", u)
		setChildText(s, "credentialsId", credentials)
		setChildText(s, "manageHooks", "true")
		updated = true
	}
	if updated {
		doc.Indent(2)
		xml2, err := doc.WriteToString()
		return prefix + xml2, err
	}
	return xml, nil
}

func createBitbucketCloudConfig(xml string, server *auth.ServerAuth, userAuth *auth.UserAuth, credentials string) (string, error) {
	elementName := bitbucketCloudElementName
	if strings.TrimSpace(xml) == "" {
		xml = defaultBitbucketXml
	}
	answer := xml
	doc, prefix, err := ParseXml(xml)
	if err != nil {
		return answer, err
	}
	root := doc.Root()
	servers := getChild(root, "endpoints")
	if servers == nil {
		servers = root.CreateElement("endpoints")
		root.AddChild(servers)
	}

	found := false
	updated := false
	for _, n := range servers.ChildElements() {
		if setChildText(n, "credentialsId", credentials) {
			updated = true
		}
		if setChildText(n, "manageHooks", "true") {
			updated = true
		}
		found = true
		break
	}
	if !found {
		// lets add a new element
		s := servers.CreateElement(elementName)
		servers.AddChild(s)
		setChildText(s, "credentialsId", credentials)
		setChildText(s, "manageHooks", "true")
		updated = true
	}
	if updated {
		doc.Indent(2)
		xml2, err := doc.WriteToString()
		return prefix + xml2, err
	}
	return xml, nil
}

func createBitbucketServerConfig(xml string, server *auth.ServerAuth, userAuth *auth.UserAuth, credentials string) (string, error) {
	elementName := bitbucketServerElementName
	u := server.URL
	if strings.TrimSpace(xml) == "" {
		xml = defaultBitbucketXml
	}
	answer := xml
	doc, prefix, err := ParseXml(xml)
	if err != nil {
		return answer, err
	}
	root := doc.Root()
	servers := getChild(root, "endpoints")
	if servers == nil {
		servers = root.CreateElement("endpoints")
		root.AddChild(servers)
	}

	found := false
	updated := false
	for _, n := range servers.ChildElements() {
		if getChildText(n, "serverUrl") == u {
			if setChildText(n, "credentialsId", credentials) {
				updated = true
			}
			found = true
			break
		}
	}
	if !found {
		// lets add a new element
		s := servers.CreateElement(elementName)
		servers.AddChild(s)
		setChildText(s, "displayName", server.Name)
		setChildText(s, "serverUrl", u)
		setChildText(s, "credentialsId", credentials)
		setChildText(s, "manageHooks", "true")
		updated = true
	}
	if updated {
		doc.Indent(2)
		xml2, err := doc.WriteToString()
		return prefix + xml2, err
	}
	return xml, nil

}

func setChildText(node *etree.Element, childName string, value string) bool {
	answer := false
	child := getChild(node, childName)
	if child == nil {
		child = node.CreateElement(childName)
		answer = true
		node.AddChild(child)
	}
	if child.Text() != value {
		child.SetText(value)
		answer = true
	}
	return answer
}

func getChild(node *etree.Element, childName string) *etree.Element {
	children := node.ChildElements()
	for _, child := range children {
		if child != nil && child.Tag == childName {
			return child
		}
	}
	return nil
}

func getChildText(node *etree.Element, childName string) string {
	child := getChild(node, childName)
	if child != nil {
		return child.Text()
	}
	return ""
}
