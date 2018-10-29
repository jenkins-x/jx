package extensions_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/pkg/extensions"

	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

const (
	binDirNs   = "jx-test"
	port       = "0"
	host       = "localhost"
	name       = "jx-test.test-plugin"
	version    = "0.0.1"
	testString = "Testing123"
)

func TestEnsurePluginInstalled(t *testing.T) {
	// Remove any existing cruft
	testPluginBinDir, err := util.PluginBinDir(binDirNs)
	assert.NoError(t, err, "Error getting plugin bin dir for namespace jx-test")
	err = os.RemoveAll(testPluginBinDir)
	assert.NoError(t, err, "Error removing existing plugin bin %s dir for namespace jx-test", binDirNs)
	// Get the dir again will cause it to be created
	testPluginBinDir, err = util.PluginBinDir(binDirNs)
	assert.NoError(t, err, "Error getting plugin bin dir for namespace jx-test")
	srv, port := serveTestScript(t)
	defer func() {
		err = srv.Close()
		assert.NoError(t, err, "Error getting plugin bin dir for namespace jx-test")
	}()
	testPlugin := jenkinsv1.Plugin{
		ObjectMeta: v1.ObjectMeta{
			Namespace: binDirNs,
		},
		Spec: jenkinsv1.PluginSpec{
			Description: "Test Plugin",
			Binaries: []jenkinsv1.Binary{
				jenkinsv1.Binary{
					Url:    fmt.Sprintf("http://%s:%d/jx-test", host, port),
					Goarch: "amd64",
					Goos:   "Windows",
				},
				jenkinsv1.Binary{
					Url:    fmt.Sprintf("http://%s:%d/jx-test", host, port),
					Goarch: "amd64",
					Goos:   "Darwin",
				},
				jenkinsv1.Binary{
					Url:    fmt.Sprintf("http://%s:%d/jx-test", host, port),
					Goarch: "amd64",
					Goos:   "Linux",
				},
			},
			Version:    version,
			Name:       name,
			SubCommand: "test-plugin",
			Group:      "Test Plugins",
		},
	}
	path, err := extensions.EnsurePluginInstalled(testPlugin)
	assert.NoError(t, err, "Error ensuring plugin is installed")
	assert.EqualValues(t, filepath.Join(testPluginBinDir, fmt.Sprintf("%s-%s", name, version)), path, "Actual path is not equal to expected path")
	cmd := util.Command{
		Name: path,
	}
	res, err := cmd.RunWithoutRetry()
	assert.NoError(t, err, "Error running plugin")
	assert.EqualValues(t, res, testString)
}

func serveTestScript(t *testing.T) (*http.Server, int) {

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		panic(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	srv := &http.Server{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "#!/bin/sh\necho %s\n", testString)
	})
	go func() {
		if err := srv.Serve(listener); err != nil {
			assert.NoError(t, err, "Error starting HTTP server to serve test plugin script")
		}
	}()
	return srv, port
}
