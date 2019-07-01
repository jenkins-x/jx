package jenkins

import "github.com/jenkins-x/jx/pkg/gits"

// BranchPattern returns the branch patterns based on git provider kind
func BranchPattern(gitKind string) string {
	switch gitKind {
	case gits.KindBitBucketCloud, gits.KindBitBucketServer:
		return BranchPatternMatchEverything
	default:
		return BranchPatternMasterPRsAndFeatures
	}
}
