package kube

const (
	// ChartGitea the default name of the gitea chart
	ChartGitea = "jenkins-x/gitea"

	// ServiceJenkins is the name of the Jenkins Service
	ServiceJenkins = "jenkins"

	// ServiceChartMuseum the service name of the Helm Chart Museum service
	ServiceChartMuseum = "jenkins-x-chartmuseum"

	// the git credentials secret
	SecretJenkinsGitCredentials = "jenkins-git-credentials"

	// LocalHelmRepoName is the default name of the local chart repository where CI / CD releases go to
	LocalHelmRepoName = "releases"

	DefaultEnvironmentGitRepoURL = "https://github.com/jenkins-x/default-environment-charts.git"
)
