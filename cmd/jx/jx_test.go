// +build covered_binary

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/cmd/jx/codecov"

	"github.com/jenkins-x/jx/pkg/log"

	"github.com/pborman/uuid"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx/cmd/jx/app"
)

func TestSystem(t *testing.T) {
	disableCoverage := os.Getenv("COVER_JX_BINARY") == "false"
	if !disableCoverage {
		disableSelfUpload := os.Getenv("COVER_SELF_UPLOAD") == "false"
		alreadyWrapped := os.Getenv("COVER_WRAPPED") == "true"
		coverprofileArgFound := false
		args := make([]string, 0)
		strippedArgs := make([]string, 0)
		for i, arg := range os.Args {
			if i > 0 && len(strippedArgs) > 0 && !strings.HasPrefix(arg, "-") && strings.HasPrefix(os.Args[i-1], "-") && os.Args[i-1] == strippedArgs[len(strippedArgs)-1] && !strings.Contains(os.Args[i-1], "=") {
				// This is an argument for the previous string
				strippedArgs = append(strippedArgs, arg)
			} else if !strings.HasPrefix(arg, "-test.") {
				args = append(args, arg)
			} else {
				strippedArgs = append(strippedArgs, arg)
			}
			if strings.HasPrefix(arg, "-test.coverprofile") {
				coverprofileArgFound = true
			}
		}
		if !alreadyWrapped && (!disableSelfUpload || !coverprofileArgFound) {
			// We need to wrap our own execution
			var outFile string
			var id string
			if !coverprofileArgFound {
				reportsDir := os.Getenv("REPORTS_DIR")
				if reportsDir == "" {
					reportsDir = filepath.Join("build", "reports")
				}
				err := os.MkdirAll(reportsDir, 0700)
				if err != nil {
					log.Errorf("Error making reports directory %s %v", reportsDir, err)
				}
				id = "jx"
				for _, arg := range args[1:] {
					if strings.HasPrefix(arg, "-") {
						break
					}
					id = fmt.Sprintf("%s_%s", id, arg)
				}
				args = append(args, "" /* use the zero value of the element type */)
				copy(args[2:], args[1:])
				outFile = filepath.Join(reportsDir, fmt.Sprintf("%s.%s.out", id, uuid.New()))
				args[1] = fmt.Sprintf("-test.coverprofile=%s", outFile)
			} else if !disableSelfUpload {
				// TODO support this
				log.Errorf("Self upload is not supported if -test.coverprofile is specified. Disabling it.")
				disableSelfUpload = true
			}
			cmd := util.Command{
				Env: map[string]string{
					"COVER_WRAPPED": "true",
				},
				Args: args[1:],
				Name: os.Args[0],
				Out:  os.Stdout,
				In:   os.Stdin,
				Err:  os.Stderr,
			}
			_, err := cmd.RunWithoutRetry()
			if !disableSelfUpload {
				if os.Getenv("CODECOV_TOKEN") == "" {
					log.Errorf("cannot upload to codecov because CODECOV_TOKEN environment variable is not set")
				} else {
					err := uploadToCodecov(outFile, id)
					if err != nil {
						log.Errorf("cannot upload to codecov because %v", err)
					}
				}

			}
			if err != nil {
				log.Error(err.Error())
				os.Exit(1)
			}
			os.Exit(0)
		} else {
			// Purposefully ignore errors from app.Run as we are checking coverage
			err := app.Run(args)
			// the assert.NoError defers the error reporting until after the coverage is written out
			assert.NoError(t, err, "error executing jx")
		}
	} else {
		main()
	}
}

func uploadToCodecov(outFile string, name string) error {
	script, err := downloadCodecovUploader()
	if err != nil {
		return errors.WithStack(err)
	}
	err = os.Chmod(script, 0700)
	if err != nil {
		return errors.Wrapf(err, "making %s executable", script)
	}
	args := []string{
		"-Z",
		"-f",
		outFile,
		"-n",
		name,
	}
	if codecov.Flag != "" {
		args = append(args, "-F", codecov.Flag)
	}
	if codecov.BuildNumber != "" {
		args = append(args, "-b", codecov.BuildNumber)
	}
	if codecov.PullRequestNumber != "" {
		args = append(args, "-P", codecov.PullRequestNumber)
	}
	if codecov.Tag != "" {
		args = append(args, "-T", codecov.Tag)
	}
	cmd := util.Command{
		Name: script,
		Env: map[string]string{
			"DOCKER_REPO":   codecov.Slug,
			"SOURCE_COMMIT": codecov.Sha,
			"SOURCE_BRANCH": codecov.Branch,
		},
		Args: args,
	}
	out, err := cmd.RunWithoutRetry()
	if err != nil {
		log.Errorf("Running %s", cmd.String())
		log.Errorf(out)
		return errors.Wrapf(err, "error uploading coverage to codecov.io")

	}
	return nil
}

func downloadCodecovUploader() (string, error) {
	script, err := ioutil.TempFile("", "codecov")
	defer script.Close()
	if err != nil {
		return "", errors.Wrapf(err, "creating tempfile")
	}
	err = downloadFile(script, "https://codecov.io/bash")
	if err != nil {
		return "", errors.Wrapf(err, "downloading codecov uploader")
	}
	return script.Name(), nil
}

func downloadFile(out *os.File, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
