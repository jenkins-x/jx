package boot_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd/boot"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBootNamespace(t *testing.T) {
	t.Parallel()

	dir := "test_data"
	projectConfig, _, err := config.LoadProjectConfig(dir)
	require.NoError(t, err, "could not load Jenkins X Pipeline")

	requirements, _, err := config.LoadRequirementsConfig(dir)
	require.NoError(t, err, "could not load Jenkins X Requirements")

	ns := boot.FindBootNamespace(projectConfig, requirements)

	assert.Equal(t, "jx", ns, "FindBootNamespace")
}
