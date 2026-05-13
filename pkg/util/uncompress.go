package util

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UncompressArtifact reads a downloaded artifact archive (.tar.gz or .zip) and
// extracts the specified binary into memory, returning an io.Reader containing its content.
func UncompressArtifact(artifactPath, url, binaryName string) (io.Reader, error) {
	if strings.HasSuffix(url, ".zip") {
		return uncompressZip(artifactPath, binaryName)
	} else if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		return uncompressTarGz(artifactPath, binaryName)
	}
	return nil, fmt.Errorf("unsupported archive format for %s", url)
}

func uncompressZip(artifactPath, binaryName string) (io.Reader, error) {
	r, err := zip.OpenReader(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file %s: %w", artifactPath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base == binaryName || base == binaryName+".exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s inside zip: %w", f.Name, err)
			}

			content, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s inside zip: %w", f.Name, err)
			}
			return bytes.NewReader(content), nil
		}
	}
	return nil, fmt.Errorf("binary %s not found in zip archive", binaryName)
}

func uncompressTarGz(artifactPath, binaryName string) (io.Reader, error) {
	f, err := os.Open(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tar.gz file %s: %w", artifactPath, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader for %s: %w", artifactPath, err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		if header.Typeflag == tar.TypeReg {
			base := filepath.Base(header.Name)
			if base == binaryName || base == binaryName+".exe" {
				content, err := io.ReadAll(tr)
				if err != nil {
					return nil, fmt.Errorf("failed to read file %s inside tar.gz: %w", header.Name, err)
				}
				return bytes.NewReader(content), nil
			}
		}
	}
	return nil, fmt.Errorf("binary %s not found in tar.gz archive", binaryName)
}
