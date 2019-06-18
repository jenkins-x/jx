package opts

const (
	MinimumMavenDeployVersion = "2.8.2"

	MasterBranch         = "master"
	DefaultGitIgnoreFile = `
.project
.classpath
.idea
.cache
.DS_Store
*.im?
target
work
`

	// DefaultIngressNamesapce default namespace fro ingress controller
	DefaultIngressNamesapce = "kube-system"
	// DefaultIngressServiceName default name for ingress controller service and deployment
	DefaultIngressServiceName = "jxing-nginx-ingress-controller"
)
