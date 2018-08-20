package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepWaitForArtifact(t *testing.T) {
	t.Parallel()
	options := &StepWaitForArtifactOptions{
		RepoURL:    defaultMavenCentralRepo,
		GroupId:    "io.jenkins.updatebot",
		ArtifactId: "updatebot-core",
		Version:    "1.1.10",
		Extension:  "pom",
	}

	err := options.Run()
	assert.NoError(t, err)

	options = &StepWaitForArtifactOptions{
		RepoURL:    defaultMavenCentralRepo,
		GroupId:    "io.jenkins.updatebot",
		ArtifactId: "does-not-exist",
		Version:    "1.1.10",
		Extension:  "pom",
		Timeout:    "1s",
		PollTime:   "100ms",
	}

	err = options.Run()
	assert.Errorf(t, err, "Should have failed to find the artifact")
}
