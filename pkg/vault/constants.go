package vault

const (
	// SystemVaultName name of the system vault used by the jenkins-x platfrom
	SystemVaultName = "jx-vault"
	// GitOpsSecretsPath the path of secrets generated for GitOps
	GitOpsSecretsPath = "gitops/"
	// GitOpsTemplatesPath the path of gitops templates secrets
	GitOpsTemplatesPath = "templates/"
	// AdminSecretsPath the path of admin secrets
	AdminSecretsPath = "admin/"
	// AuthSecretsPath the path of auth secrets
	AuthSecretsPath = "auth/"
)

// AdminSecret type for a vault admin secret
type AdminSecret string

const (
	// JenkinsAdminSecret the secret name for Jenkins admin password
	JenkinsAdminSecret = "jenkins"
	// NexusAdminSecret the secret name for Nexus credentials
	NexusAdminSecret = "nexus"
	// ChartmuseumAdminSecret the secret name for ChartMuseum credentials
	ChartmuseumAdminSecret = "chartmuseum"
	// GrafanaAdminSecret the secret name for Grafana credentials
	GrafanaAdminSecret = "grafana"
	// IngressAdminSecret the secret name for Ingress basic authentication
	IngressAdminSecret = "ingress"
)
