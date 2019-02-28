package builds_test

import (
	"github.com/jenkins-x/jx/pkg/builds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetBuildNumberFromLabelsFileData(t *testing.T) {
	t.Parallel()

	assertBuildNumberFromLabelsData(t, `build.knative.dev/buildName="jstrachan-mynodething-master-24-build"`, "24")
	assertBuildNumberFromLabelsData(t, `build.knative.dev/buildName="jstrachan-mynodething-master-12"`, "12")
	assertBuildNumberFromLabelsData(t, `build-number="45"`, "45")
}

func assertBuildNumberFromLabelsData(t *testing.T, text string, expected string) {
	m := builds.LoadDownwardAPILabels(text)
	require.NotNil(t, "could not load map from downward API text: %s", text)
	actual := builds.GetBuildNumberFromLabelsFileData(m)
	assert.Equal(t, expected, actual, "GetBuildNumberFromLabelsFileData() with data %s", text)
}
