package kube

const (
	// ChartGitea the default name of the gitea chart
	ChartGitea = "jenkins-x/gitea"

	// ChartCDX the default name of the CDX chart
	ChartCDX = "jenkins-x/cdx"

	// ServiceJenkins is the name of the Jenkins Service
	ServiceJenkins = "jenkins"

	// SeriviceCDX the service name of the Helm Chart Museum service
	ServiceCDX = "cdx-cdx"

	// ServiceChartMuseum the service name of the Helm Chart Museum service
	ServiceChartMuseum = "jenkins-x-chartmuseum"

	// ServiceKubernetesDashboard the kubernetes dashboard
	ServiceKubernetesDashboard = "jenkins-x-kubernetes-dashboard"

	// the git credentials secret
	SecretJenkinsGitCredentials = "jenkins-git-credentials"

	// the issue tracker credentials secret
	SecretJenkinsIssueCredentials = "jenkins-issues-"

	// LocalHelmRepoName is the default name of the local chart repository where CI / CD releases go to
	LocalHelmRepoName = "releases"

	DefaultEnvironmentGitRepoURL = "https://github.com/jenkins-x/default-environment-charts.git"
)

var (
	AddonCharts = map[string]string{
		"gitea":      ChartGitea,
		"cdx":        ChartCDX,
		"prometheus": "stable/prometheus",
		"grafana":    "stable/grafana",
	}
)
