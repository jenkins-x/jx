package syntax

const (
	// TektonAPIVersion the APIVersion for using Tekton
	TektonAPIVersion = "tekton.dev/v1alpha1"

	// LabelStageName - the name for the label that will have the stage name on the Task.
	LabelStageName = "jenkins.io/task-stage-name"

	// LabelPipelineFromYaml - if true, the CRD was created by translation from YAML, rather than from a build pack.
	LabelPipelineFromYaml = "jenkins.io/pipeline-from-yaml"
)
