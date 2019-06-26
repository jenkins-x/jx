package tekton

const (
	// LastBuildNumberAnnotationPrefix used to annotate SourceRepository with the latest build number for a branch
	LastBuildNumberAnnotationPrefix = "jenkins.io/last-build-number-for-"

	// LabelBranch is the label added to Tekton CRDs for the branch being built.
	LabelBranch = "branch"

	// LabelBuild is the label added to Tekton CRDs for the build number.
	LabelBuild = "build"

	// LabelContext is the label added to Tekton CRDs for the context being built.
	LabelContext = "context"

	// LabelOwner is the label added to Tekton CRDs for the owner of the repository being built.
	LabelOwner = "owner"

	// LabelRepo is the label added to Tekton CRDs for the repository being built.
	LabelRepo = "repo"
)
