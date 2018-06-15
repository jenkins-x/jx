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

package client

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

func (c *SonobuoyClient) RetrieveResults(cfg *RetrieveConfig) (io.Reader, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	restClient := client.CoreV1().RESTClient()
	req := restClient.Post().
		Resource("pods").
		Name(config.MasterPodName).
		Namespace(cfg.Namespace).
		SubResource("exec").
		Param("container", config.MasterContainerName)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: config.MasterContainerName,
		Command:   []string{"tar", "cf", "-", config.MasterResultsPath},
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(c.RestConfig, "POST", req.URL())
	if err != nil {
		return nil, err
	}
	reader, writer := io.Pipe()
	go func(writer *io.PipeWriter) {
		defer writer.Close()
		err = executor.Stream(remotecommand.StreamOptions{
			Stdout: writer,
			Stderr: os.Stderr,
			Tty:    false,
		})
		if err != nil {
			// Since this function returns an io.Reader to the consumer and does
			// not buffer the entire (potentially large) output, RetrieveResults
			// has to return the reader first to be read from. This means we
			// either lose this error (easy) or provide a significantly more
			// complex error mechanism for the consumer (hard).
			logrus.Error(err)
		}
	}(writer)

	return reader, nil
}

/** Everything below this marker has been copy/pasta'd from k8s/k8s. The only modification is exporting UntarAll **/

// UntarAll expects a reader that contains tar'd data. It will untar the contents of the reader and write
// the output into destFile under the prefix, prefix.
func UntarAll(reader io.Reader, destFile, prefix string) error {
	entrySeq := -1

	// TODO: use compression here?
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		entrySeq++
		mode := header.FileInfo().Mode()
		outFileName := path.Join(destFile, header.Name[len(prefix):])
		baseName := path.Dir(outFileName)
		if err := os.MkdirAll(baseName, 0755); err != nil {
			return err
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(outFileName, 0755); err != nil {
				return err
			}
			continue
		}

		// handle coping remote file into local directory
		if entrySeq == 0 && !header.FileInfo().IsDir() {
			exists, err := dirExists(outFileName)
			if err != nil {
				return err
			}
			if exists {
				outFileName = filepath.Join(outFileName, path.Base(header.Name))
			}
		}

		if mode&os.ModeSymlink != 0 {
			err := os.Symlink(header.Linkname, outFileName)
			if err != nil {
				return err
			}
		} else {
			outFile, err := os.Create(outFileName)
			if err != nil {
				return err
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			if err := outFile.Close(); err != nil {
				return err
			}
		}
	}

	if entrySeq == -1 {
		//if no file was copied
		errInfo := fmt.Sprintf("error: %s no such file or directory", prefix)
		return errors.New(errInfo)
	}
	return nil
}

// dirExists checks if a path exists and is a directory.
func dirExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil && fi.IsDir() {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
