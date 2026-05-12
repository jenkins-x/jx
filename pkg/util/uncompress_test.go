package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUncompressZip(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	binaryName := "mybin.exe"
	content := []byte("binary data")

	// Create a zip file
	f, err := os.Create(zipPath)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	fWriter, err := w.Create(binaryName)
	require.NoError(t, err)
	_, err = fWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	// Test uncompress
	reader, err := UncompressArtifact(zipPath, "http://example.com/test.zip", "mybin")
	require.NoError(t, err)

	extracted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, extracted)
}

func TestUncompressTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test.tar.gz")
	binaryName := "mybin"
	content := []byte("binary data")

	// Create a tar.gz file
	f, err := os.Create(tarPath)
	require.NoError(t, err)
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: "some/path/" + binaryName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	// Test uncompress
	reader, err := UncompressArtifact(tarPath, "http://example.com/test.tar.gz", "mybin")
	require.NoError(t, err)

	extracted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, extracted)
}
