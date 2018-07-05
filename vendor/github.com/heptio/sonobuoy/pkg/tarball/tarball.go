/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tarball

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
)

// DecodeTarball takes a reader and a base directory, and extracts a gzipped tarball rooted on
// the given directory. If there is an error, the imput may only be partially consumed.
// At the moment, the tarball decoder only supports directories, regular files and symlinks.
func DecodeTarball(reader io.Reader, baseDir string) error {
	gzStream, err := gzip.NewReader(reader)
	if err != nil {
		return errors.Wrap(err, "couldn't uncompress reader")
	}
	defer gzStream.Close()

	tarchive := tar.NewReader(gzStream)
	for {
		header, err := tarchive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "couldn't opening tarball from gzip")
		}
		name := path.Clean(header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path.Join(baseDir, name), os.FileMode(header.Mode)); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (mkdir)")
			}
		case tar.TypeReg, tar.TypeRegA:
			filePath := path.Join(baseDir, name)
			// Directory should come first, but some tarballes are malformed
			if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (mkdir)")
			}
			file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return errors.Wrap(err, "error decoding tarball for result (open)")
			}
			if _, err := io.CopyN(file, tarchive, header.Size); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (copy)")
			}
		case tar.TypeSymlink:
			filePath := path.Join(baseDir, name)
			// Directory should come first, but some tarballes are malformed
			if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
				return errors.Wrapf(err, "error decoding tarball for result (mkdir)")
			}
			if err := os.Symlink(
				path.Join(baseDir, path.Clean(header.Linkname)),
				path.Join(baseDir, name),
			); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (ln)")
			}
		default:
		}
	}

	return nil
}
