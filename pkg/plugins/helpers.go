package plugins

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"

	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/httphelpers"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	jenkinsxOrganisation        = "jenkins-x"
	jenkinsxPluginsOrganisation = "jenkins-x-plugins"

	// OctantPluginName the default name of the octant plugin
	OctantPluginName = "octant"

	// OctantJXPluginName the name of the octant-jx plugin
	OctantJXPluginName = "octant-jx"

	// OctantJXOPluginName the name of the octant-jxo plugin
	OctantJXOPluginName = "octant-jxo"
)

// GetJXPlugin returns the path to the locally installed jx plugin
func GetJXPlugin(name, version string) (string, error) {
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := extensions.CreateJXPlugin(jenkinsxOrganisation, name, version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// GetOctantBinary returns the path to the locally installed octant plugin
func GetOctantBinary(version string) (string, error) {
	if version == "" {
		version = OctantVersion
	}
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := CreateOctantPlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateOctantPlugin creates the helm 3 plugin
func CreateOctantPlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {
		kind := strings.ToLower(p.Goarch)
		if strings.HasSuffix(kind, "64") {
			kind = "64bit"
		}
		goos := p.Goos
		if goos == "Darwin" {
			goos = "macOS"
		}
		return fmt.Sprintf("https://github.com/vmware-tanzu/octant/releases/download/v%s/octant_%s_%s-%s.%s", version, version, goos, kind, p.Extension())
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: OctantPluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "octant",
			Binaries:    binaries,
			Description: "octant binary",
			Name:        OctantPluginName,
			Version:     version,
		},
	}
	return plugin
}

// GetOctantJXBinary returns the path to the locally installed octant-jx extension
func GetOctantJXBinary(version string) (string, error) {
	if version == "" {
		version = OctantJXVersion
	}
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := CreateOctantJXPlugin(version)
	return extensions.EnsurePluginInstalled(plugin, pluginBinDir)
}

// CreateOctantJXPlugin creates the helm 3 plugin
func CreateOctantJXPlugin(version string) jenkinsv1.Plugin {
	binaries := extensions.CreateBinaries(func(p extensions.Platform) string {
		return fmt.Sprintf("https://github.com/jenkins-x-plugins/octant-jx/releases/download/v%s/octant-jx-%s-%s.%s", version, strings.ToLower(p.Goos), strings.ToLower(p.Goarch), p.Extension())
	})

	plugin := jenkinsv1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: OctantJXPluginName,
		},
		Spec: jenkinsv1.PluginSpec{
			SubCommand:  "octant-jx",
			Binaries:    binaries,
			Description: "octant plugin for Jenkins X",
			Name:        OctantJXPluginName,
			Version:     version,
		},
	}
	return plugin
}

// GetOctantJXOBinary returns the path to the locally installed helmAnnotate extension
func GetOctantJXOBinary(version string) (string, error) {
	if version == "" {
		version = OctantJXVersion
	}
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return "", err
	}
	plugin := CreateOctantJXOPlugin(version)
	aliasFileName := "ha.tar.gz"
	if runtime.GOOS == "windows" {
		aliasFileName = "ha.zip"
	}
	return extensions.EnsurePluginInstalledForAliasFile(plugin, pluginBinDir, aliasFileName)
}

// CreateOctantJXOPlugin creates the octant-ojx plugin
func CreateOctantJXOPlugin(version string) jenkinsv1.Plugin {
	plugin := CreateOctantJXPlugin(version)
	plugin.Name = OctantJXOPluginName
	plugin.Spec.Name = OctantJXOPluginName
	plugin.Spec.SubCommand = OctantJXOPluginName
	return plugin
}

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// InstallStandardPlugin makes sure that latest version of plugin is installed and returns the path to the binary
func InstallStandardPlugin(dir, name string) (string, error) {
	u := "https://api.github.com/repos/jenkins-x-plugins/" + name + "/releases/latest"

	client := httphelpers.GetClient()
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create http request for %s", u)
	}
	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			return "", errors.Wrapf(err, "failed to GET endpoint %s with status %s", u, resp.Status)
		}
		return "", errors.Wrapf(err, "failed to GET endpoint %s", u)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read response from %s", u)
	}

	release := &githubRelease{}
	err = json.Unmarshal(body, release)
	if err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal release from %s", u)
	}
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if latestVersion == "" {
		return "", fmt.Errorf("can't find latest version of plugin: %s", body)
	}

	plugin := extensions.CreateJXPlugin("jenkins-x-plugins", strings.TrimPrefix(name, "jx-"), latestVersion)
	if err != nil {
		return "", err
	}
	return extensions.EnsurePluginInstalled(plugin, dir)
}
