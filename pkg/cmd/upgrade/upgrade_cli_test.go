// +build unit

package upgrade

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/blang/semver"
	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/brew"
	"github.com/jenkins-x/jx/v2/pkg/cmd/create/options"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/extensions"
	"github.com/jenkins-x/jx/v2/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var sampleBrewInfo = `[  
   {  
      "name":"jx",
      "full_name":"jenkins-x/jx/jx",
      "oldname":null,
      "aliases":[  

      ],
      "versioned_formulae":[  

      ],
      "desc":"A tool to install and interact with Jenkins X on your Kubernetes cluster.",
      "homepage":"https://jenkins-x.github.io/jenkins-x-website/",
      "versions":{  
         "stable":"2.0.181",
         "devel":null,
         "head":null,
         "bottle":false
      },
      "revision":0,
      "version_scheme":0,
      "bottle":{  

      },
      "keg_only":false,
      "bottle_disabled":false,
      "options":[  

      ],
      "build_dependencies":[  

      ],
      "dependencies":[  

      ],
      "recommended_dependencies":[  

      ],
      "optional_dependencies":[  

      ],
      "requirements":[  

      ],
      "conflicts_with":[  

      ],
      "caveats":null,
      "installed":[  
         {  
            "version":"2.0.181",
            "used_options":[  

            ],
            "built_as_bottle":false,
            "poured_from_bottle":false,
            "runtime_dependencies":[  

            ],
            "installed_as_dependency":false,
            "installed_on_request":true
         }
      ],
      "linked_keg":"2.0.181",
      "pinned":false,
      "outdated":false
   }
]`

func TestLatestJxBrewVersion(t *testing.T) {
	version, err := brew.LatestJxBrewVersion(sampleBrewInfo)
	assert.NoError(t, err)
	assert.Equal(t, "2.0.181", version)
}

func TestNeedsUpgrade(t *testing.T) {
	type testData struct {
		current               string
		latest                string
		expectedUpgradeNeeded bool
		expectedMessage       string
	}

	testCases := []testData{
		{
			"1.0.0", "1.0.0", false, "You are already on the latest version of jx 1.0.0\n",
		},
		{
			"1.0.0", "1.0.1", true, "",
		},
		{
			"1.0.0", "0.0.99", true, "",
		},
	}

	o := UpgradeCLIOptions{}
	for _, data := range testCases {
		currentVersion, _ := semver.New(data.current)
		latestVersion, _ := semver.New(data.latest)
		actualMessage := log.CaptureOutput(func() {
			actualUpgradeNeeded := o.needsUpgrade(*currentVersion, *latestVersion)
			assert.Equal(t, data.expectedUpgradeNeeded, actualUpgradeNeeded, fmt.Sprintf("Unexpected upgrade flag for %v", data))
		})
		assert.Equal(t, data.expectedMessage, actualMessage, fmt.Sprintf("Unexpected message for %v", data))
	}
}

func TestVersionCheckWhenCurrentVersionIsGreaterThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	version.Map["version"] = "1.4.0"
	opts := &UpgradeCLIOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersionCheckWhenCurrentVersionIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.3"
	opts := &UpgradeCLIOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersionCheckWhenCurrentVersionIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 3, Patch: 153}
	version.Map["version"] = "1.0.0"
	opts := &UpgradeCLIOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.True(t, update, "should update")
}

func TestVersionCheckWhenCurrentVersionIsEqualToReleaseVersionWithPatch(t *testing.T) {
	prVersions := []semver.PRVersion{}
	prVersions = append(prVersions, semver.PRVersion{VersionStr: "dev"})
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3, Pre: prVersions, Build: []string(nil)}
	version.Map["version"] = "1.2.3"
	opts := &UpgradeCLIOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersionCheckWhenCurrentVersionWithPatchIsEqualToReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.3-dev+6a8285f4"
	opts := &UpgradeCLIOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestVersionCheckWhenCurrentVersionWithPatchIsLessThanReleaseVersion(t *testing.T) {
	jxVersion := semver.Version{Major: 1, Minor: 2, Patch: 3}
	version.Map["version"] = "1.2.2-dev+6a8285f4"
	opts := &UpgradeCLIOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	update, err := opts.ShouldUpdate(jxVersion)
	assert.NoError(t, err, "should check version without failure")
	assert.False(t, update, "should not update")
}

func TestUpgradeBinaryPlugins(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	opts := &UpgradeCLIOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: &opts.CommonOptions{},
		},
	}
	dummyPluginURL := "https://raw.githubusercontent.com/jenkins-x/jx/master/hack/gofmt.sh"
	ns := "jx"
	pluginName := "jx-my-plugin"
	pluginVersion := "1.2.3"
	jxClient := fake.NewSimpleClientset(
		&v1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pluginName,
				Namespace: ns,
				Labels: map[string]string{
					extensions.PluginCommandLabel: pluginName,
				},
			},
			Spec: v1.PluginSpec{
				Name:       pluginName,
				SubCommand: "my plugin",
				Group:      "",
				Binaries: []v1.Binary{
					{
						URL:    dummyPluginURL,
						Goarch: "amd64",
						Goos:   "Windows",
					},
					{
						URL:    dummyPluginURL,
						Goarch: "amd64",
						Goos:   "Darwin",
					},
					{
						URL:    dummyPluginURL,
						Goarch: "amd64",
						Goos:   "Linux",
					},
				},
				Description: "my awesome plugin extension",
				Version:     pluginVersion,
			},
		})
	opts.SetJxClient(jxClient)
	opts.SetDevNamespace(ns)

	oldJXHome := os.Getenv("JX_HOME")
	os.Setenv("JX_HOME", tmpDir)
	defer os.Setenv("JX_HOME", oldJXHome)

	t.Logf("downloading plugins to JX_HOME %s\n", tmpDir)

	err = opts.UpgradeBinaryPlugins()
	require.NoError(t, err, "should not fail upgrading the binary plugins")
	assert.FileExists(t, filepath.Join(tmpDir, "plugins", ns, "bin", pluginName+"-"+pluginVersion))
}
