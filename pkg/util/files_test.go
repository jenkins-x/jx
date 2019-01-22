package util_test

import (
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	"sort"
	"testing"
)

func TestGlobFiles(t *testing.T) {
	t.Parallel()

	files := []string{}
	fn := func(name string) error {
		if util.StringArrayIndex(files, name) < 0 {
			files = append(files, name)
		}
		return nil
	}

	/*	pwd, err := os.Getwd()
		require.NoError(t, err)
		t.Logf("Current dir is %s\n", pwd)
	*/
	err := util.GlobAllFiles("", "test_data/glob_test/*", fn)
	require.NoError(t, err)

	for _, f := range files {
		t.Logf("Processed file %s\n", f)
	}

	sort.Strings(files)

	t.Logf("Found %d files\n", len(files))

	expected := []string{
		"test_data/glob_test/artifacts/goodbye.txt",
		"test_data/glob_test/hello.txt",
	}
	
	assert.Equal(t, expected, files, "globbed files")
}
