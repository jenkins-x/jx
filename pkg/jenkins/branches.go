package jenkins

import "github.com/jenkins-x/jx/v2/pkg/gits"

func BranchPattern(gitKind string) string {
	switch gitKind {
	case gits.KindBitBucketCloud, gits.KindBitBucketServer:
		return BranchPatternMatchEverything
	default:
		return BranchPatternMasterPRsAndFeatures
	}
}
