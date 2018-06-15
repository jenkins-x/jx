package kube

const (
	// ChartAmbassador the default chart for ambassador
	ChartAmbassador = "datawire/ambassador"

	// ChartAnchore the default chart for the Anchore plugin
	ChartAnchore = "stable/anchore-engine"

	// ChartCloudBees the default name of the CloudBees addon chart
	ChartCloudBees = "cb/cdx"

	// ChartExposecontrollerService the default name of the Exposecontroller Service chart for Edit environments
	ChartExposecontrollerService = "jenkins-x/exposecontroller-service"

	// ChartAnchore the default chart for the Anchore plugin
	ChartPipelineEvent = "jenkins-x/pipeline-events-addon"

	// ChartGitea the default name of the gitea chart
	ChartGitea = "jenkins-x/gitea"

	// ChartIstio the default chart for the Istio chart
	ChartIstio = "install/kubernetes/helm/istio"

	// ChartKubeless the default chart for kubeless
	ChartKubeless = "incubator/kubeless"

	// ServiceJenkins is the name of the Jenkins Service
	ServiceJenkins = "jenkins"

	// SecretJenkins is the name of the Jenkins secret
	SecretJenkins = "jenkins"

	// ServiceCloudBees the service name of the CloudBees app for Kubernetes
	ServiceCloudBees = "cb-cdx"

	// ServiceChartMuseum the service name of the Helm Chart Museum service
	ServiceChartMuseum = "jenkins-x-chartmuseum"

	// ServiceKubernetesDashboard the kubernetes dashboard
	ServiceKubernetesDashboard = "jenkins-x-kubernetes-dashboard"

	// SecretJenkinsGitCredentials the git credentials secret
	SecretJenkinsGitCredentials = "jenkins-git-credentials"

	// SecretJenkinsChartMuseum the chart museum secret
	SecretJenkinsChartMuseum = "jenkins-x-chartmuseum"

	// SecretJenkinsReleaseGPG the GPG secrets for doing releases
	SecretJenkinsReleaseGPG = "jenkins-release-gpg"

	// SecretJenkinsPipelineAddonCredentials the chat credentials secret
	SecretJenkinsPipelineAddonCredentials = "jx-pipeline-addon-"

	// SecretJenkinsPipelineChatCredentials the chat credentials secret
	SecretJenkinsPipelineChatCredentials = "jx-pipeline-chat-"

	// SecretJenkinsPipelineGitCredentials the git credentials secret
	SecretJenkinsPipelineGitCredentials = "jx-pipeline-git-"

	// SecretJenkinsPipelineIssueCredentials the issue tracker credentials secret
	SecretJenkinsPipelineIssueCredentials = "jx-pipeline-issues-"

	// ConfigMapExposecontroller the name of the ConfigMap with the Exposecontroller configuration
	ConfigMapExposecontroller = "exposecontroller"

	// ConfigMapJenkinsXGitKinds the name of the ConfigMap in the development namespace that maps kinds to URLs
	ConfigMapJenkinsXGitKinds = "jenkins-x-git-kinds"

	// ConfigMapJenkinsX the name of the ConfigMap with the Jenkins configuration
	ConfigMapJenkinsX = "jenkins"

	// ConfigMapJenkinsPodTemplates is the ConfigMap containing all the Pod Templates available
	ConfigMapJenkinsPodTemplates = "jenkins-x-pod-templates"

	// LocalHelmRepoName is the default name of the local chart repository where CI/CD releases go to
	LocalHelmRepoName = "releases"

	// DeploymentExposecontrollerService the name of the Deployment for the Exposecontroller Service
	DeploymentExposecontrollerService = "exposecontroller-service"

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

	// ValueKindCVE an addon auth secret/credentials
	ValueKindCVE = "cve"

	// ValueKindCVE an addon auth PipelineEvent
	ValueKindPipelineEvent = "PipelineEvent"

	// ValueKindCVE an addon auth PipelineEvent
	ValueKindRelease = "Release"

	// ValueKindEditNamespace for edit namespace
	ValueKindEditNamespace = "editspace"

	// LabelServiceKind the label to indicate the auto Server's Kind
	LabelServiceKind = "jenkins.io/service-kind"

	// LabelCreatedBy indicates the service that created this resource
	LabelCreatedBy = "jenkins.io/created-by"

	// LabelPodTemplate the name of the pod template for a DevPod
	LabelPodTemplate = "jenkins.io/pod_template"

	// LabelDevPodName the name of a dev pod
	LabelDevPodName = "jenkins.io/devpod"

	// LabelDevPodUsername the user name owner of the DeVPod
	LabelDevPodUsername = "jenkins.io/devpod_user"

	// LabelUsername the user name owner of a namespace or resource
	LabelUsername = "jenkins.io/user"

	// ValueCreatedByJX for resources created by the Jenkins X CLI
	ValueCreatedByJX = "jx"

	// LabelCredentialsType the kind of jenkins credential for a secret
	LabelCredentialsType = "jenkins.io/credentials-type"

	// ValueCredentialTypeUsernamePassword for user password credential secrets
	ValueCredentialTypeUsernamePassword = "usernamePassword"

	// AnnotationURL indicates a service/server's URL
	AnnotationURL = "jenkins.io/url"

	// AnnotationExpose used to expose service using exposecontroller
	AnnotationExpose = "fabric8.io/expose"

	// AnnotationIngress tells exposecontroller to annotate generated ingress rule with values
	AnnotationIngress = "fabric8.io/ingress.annotations"

	// AnnotationName indicates a service/server's textual name (can be mixed case, contain spaces unlike kubernetes resources)
	AnnotationName = "jenkins.io/name"

	// AnnotationCredentialsDescription the description text for a Credential on a Secret
	AnnotationCredentialsDescription = "jenkins.io/credentials-description"

	// AnnotationWorkingDir the working directory, such as for a DevPod
	AnnotationWorkingDir = "jenkins.io/working-dir"
	// AnnotationLocalDir the local directory that is sync'd to the DevPod
	AnnotationLocalDir = "jenkins.io/local-dir"

	// SecretDataUsername the username in a Secret/Credentials
	SecretDataUsername = "username"

	// SecretDataPassword the password in a Secret/Credentials
	SecretDataPassword = "password"

	// SecretBasicAuth the name for the Jenkins X basic auth secret
	SecretBasicAuth = "jx-basic-auth"

	JenkinsAdminApiToken = "jenkins-admin-api-token"

	AUTH = "auth"
)

var (
	AddonCharts = map[string]string{
		"ambassador": ChartAmbassador,
		"anchore":    ChartAnchore,
		"cb":         ChartCloudBees,
		"gitea":      ChartGitea,
		"istio":      ChartIstio,
		"kubeless":   ChartKubeless,
		"prometheus": "stable/prometheus",
		"grafana":    "stable/grafana",
	}

	AddonServices = map[string]string{
		"anchore":         "anchore-anchore-engine",
		"pipeline-events": "jx-pipeline-events-elasticsearch-client",
	}
)
