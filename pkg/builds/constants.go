package builds

const (
	// LabelBuildName the label used on a pod for the build name
	LabelBuildName    = "build.knative.dev/buildName"
	LabelOldBuildName = "build-name"

	// LabelPipelineRunName the label used on a pod created via Build Pipeline for the build name.
	LabelPipelineRunName = "pipeline.knative.dev/pipelineRun"
)
