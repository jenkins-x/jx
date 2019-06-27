package releases

import (
	"github.com/jenkins-x/jx/pkg/gits"
)

// ReleaseDownloadCount returns the total number of downloads for the given set of releases
func ReleaseDownloadCount(releases []*gits.GitRelease) int {
	count := 0
	for _, release := range releases {
		count += release.DownloadCount
	}
	return count
}
