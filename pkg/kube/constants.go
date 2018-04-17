package kube

const (
	// ChartAmbassador the default chart for ambassador
	ChartAmbassador = "datawire/ambassador"

	// ChartAnchore the default chart for the Anchore plugin
	ChartAnchore = "stable/anchore-engine"

	// ChartCDX the default name of the CDX chart
	ChartCDX = "jenkins-x/cdx"

	// ChartGitea the default name of the gitea chart
	ChartGitea = "jenkins-x/gitea"

	// ChartKubeless the default chart for kubeless
	ChartKubeless = "incubator/kubeless"

	// ServiceJenkins is the name of the Jenkins Service
	ServiceJenkins = "jenkins"

	// SeriviceCDX the service name of the Helm Chart Museum service
	ServiceCDX = "cdx-cdx"

	// ServiceChartMuseum the service name of the Helm Chart Museum service
	ServiceChartMuseum = "jenkins-x-chartmuseum"

	// ServiceKubernetesDashboard the kubernetes dashboard
	ServiceKubernetesDashboard = "jenkins-x-kubernetes-dashboard"

	// SecretJenkinsGitCredentials the git credentials secret
	SecretJenkinsGitCredentials = "jenkins-git-credentials"

	// SecretJenkinsPipelineAddonCredentials the chat credentials secret
	SecretJenkinsPipelineAddonCredentials = "jx-pipeline-addon-"

	// SecretJenkinsPipelineChatCredentials the chat credentials secret
	SecretJenkinsPipelineChatCredentials = "jx-pipeline-chat-"

	// SecretJenkinsPipelineGitCredentials the git credentials secret
	SecretJenkinsPipelineGitCredentials = "jx-pipeline-git-"

	// SecretJenkinsPipelineIssueCredentials the issue tracker credentials secret
	SecretJenkinsPipelineIssueCredentials = "jx-pipeline-issues-"

	// ConfigMapJenkinsXGitKinds the name of the ConfigMap in the development namespace that maps kinds to URLs
	ConfigMapJenkinsXGitKinds = "jenkins-x-git-kinds"

	// ConfigMapJenkinsX the name of the ConfigMap with the Jenkins configuration
	ConfigMapJenkinsX = "jenkins"

	// LocalHelmRepoName is the default name of the local chart repository where CI/CD releases go to
	LocalHelmRepoName = "releases"

	DefaultEnvironmentGitRepoURL = "https://github.com/jenkins-x/default-environment-charts.git"

	// LabelKind to indicate the kind of auth, such as Git or Issue
	LabelKind = "jenkins.io/kind"

	// ValueKindAddon an addon auth secret/credentials
	ValueKindAddon = "addon"

	// ValueKindChat a chat auth secret/credentials
	ValueKindChat = "chat"

	// ValueKindGit a git auth secret/credentials
	ValueKindGit = "git"

	// ValueKindIssue an issue auth secret/credentials
	ValueKindIssue = "issue"

	// LabelServiceKind the label to indicate the auto Server's Kind
	LabelServiceKind = "jenkins.io/service-kind"

	// LabelCreatedBy indicates the service that created this resource
	LabelCreatedBy = "jenkins.io/created-by"

	// ValueCreatedByJX for resources created by the Jenkins X CLI
	ValueCreatedByJX = "jx"

	// LabelCredentialsType the kind of jenkins credential for a secret
	LabelCredentialsType = "jenkins.io/credentials-type"

	// ValueCredentialTypeUsernamePassword for user password credential secrets
	ValueCredentialTypeUsernamePassword = "usernamePassword"

	// AnnotationURL indicates a service/server's URL
	AnnotationURL = "jenkins.io/url"

	// AnnotationName indicates a service/server's textual name (can be mixed case, contain spaces unlike kubernetes resources)
	AnnotationName = "jenkins.io/name"

	// AnnotationCredentialsDescription the description text for a Credentian on a Secret
	AnnotationCredentialsDescription = "jenkins.io/credentials-description"

	// SecretDataUsername the username in a Secret/Credentials
	SecretDataUsername = "username"

	// SecretDataPassword the password in a Secret/Credentials
	SecretDataPassword = "password"
)

var (
	AddonCharts = map[string]string{
		"ambassador": ChartAmbassador,
		"anchore":    ChartAnchore,
		"cdx":        ChartCDX,
		"gitea":      ChartGitea,
		"kubeless":   ChartKubeless,
		"prometheus": "stable/prometheus",
		"grafana":    "stable/grafana",
	}
)
