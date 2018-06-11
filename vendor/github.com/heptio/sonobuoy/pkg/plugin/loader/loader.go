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

package loader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/daemonset"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/job"
	"github.com/heptio/sonobuoy/pkg/plugin/manifest"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

// LoadAllPlugins loads all plugins by finding plugin definitions in the given
// directory, taking a user's plugin selections, and a sonobuoy phone home
// address (host:port) and returning all of the active, configured plugins for
// this sonobuoy run.
func LoadAllPlugins(namespace, sonobuoyImage, imagePullPolicy string, searchPath []string, selections []plugin.Selection) (ret []plugin.Interface, err error) {
	pluginDefinitionFiles := make(map[string]struct{})
	for _, dir := range searchPath {
		wd, _ := os.Getwd()
		logrus.Infof("Scanning plugins in %v (pwd: %v)", dir, wd)

		// We only care about configured plugin directories that exist,
		// since we may have a broad search path.
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			logrus.Infof("Directory (%v) does not exist", dir)
			continue
		}

		files, err := findPlugins(dir)
		if err != nil {
			return []plugin.Interface{}, errors.Wrapf(err, "couldn't scan %v for plugins", dir)
		}
		for _, file := range files {
			pluginDefinitionFiles[file] = struct{}{}
		}
	}

	pluginDefinitions := []*manifest.Manifest{}
	for file := range pluginDefinitionFiles {
		definitionFile, err := loadDefinitionFromFile(file)
		if err != nil {
			return []plugin.Interface{}, errors.Wrapf(err, "couldn't load plugin definition file %v", file)
		}
		pluginDefinition, err := loadDefinition(definitionFile)
		if err != nil {
			return []plugin.Interface{}, errors.Wrapf(err, "couldn't load plugin definition for file %v", file)
		}

		pluginDefinitions = append(pluginDefinitions, pluginDefinition)
	}

	pluginDefinitions = filterPluginDef(pluginDefinitions, selections)

	plugins := []plugin.Interface{}
	for _, def := range pluginDefinitions {
		loadedPlugin, err := loadPlugin(def, namespace, sonobuoyImage, imagePullPolicy)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load plugin %v", def.SonobuoyConfig.PluginName)
		}
		plugins = append(plugins, loadedPlugin)
	}

	return plugins, nil
}

func findPlugins(dir string) ([]string, error) {
	candidates, err := ioutil.ReadDir(dir)
	if err != nil {
		return []string{}, errors.Wrapf(err, "couldn't search path %v", dir)
	}

	plugins := []string{}
	for _, candidate := range candidates {
		if candidate.IsDir() {
			continue
		}
		ext := filepath.Ext(candidate.Name())
		if ext == ".yml" || ext == ".yaml" {
			plugins = append(plugins, filepath.Join(dir, candidate.Name()))
		}
	}
	return plugins, nil
}

func loadDefinitionFromFile(file string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(file)
	return bytes, errors.Wrapf(err, "couldn't open plugin definition %v", file)
}

func loadDefinition(bytes []byte) (*manifest.Manifest, error) {
	var def manifest.Manifest
	err := kuberuntime.DecodeInto(manifest.Decoder, bytes, &def)
	return &def, errors.Wrap(err, "couldn't decode yaml for plugin definition")
}

func loadPlugin(def *manifest.Manifest, namespace, sonobuoyImage, imagePullPolicy string) (plugin.Interface, error) {
	pluginDef := plugin.Definition{
		Name:         def.SonobuoyConfig.PluginName,
		ResultType:   def.SonobuoyConfig.ResultType,
		ExtraVolumes: def.ExtraVolumes,
		Spec:         def.Spec,
	}

	switch def.SonobuoyConfig.Driver {
	case "Job":
		return job.NewPlugin(pluginDef, namespace, sonobuoyImage, imagePullPolicy), nil
	case "DaemonSet":
		return daemonset.NewPlugin(pluginDef, namespace, sonobuoyImage, imagePullPolicy), nil
	default:
		return nil, fmt.Errorf("unknown driver %q for plugin %v",
			def.SonobuoyConfig.Driver, def.SonobuoyConfig.PluginName)
	}
}

func filterPluginDef(defs []*manifest.Manifest, selections []plugin.Selection) []*manifest.Manifest {
	m := make(map[string]bool)
	for _, selection := range selections {
		m[selection.Name] = true
	}

	filtered := []*manifest.Manifest{}
	for _, def := range defs {
		if m[def.SonobuoyConfig.PluginName] {
			filtered = append(filtered, def)
		}
	}
	return filtered
}
