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

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/heptio/sonobuoy/pkg/plugin"
	pluginloader "github.com/heptio/sonobuoy/pkg/plugin/loader"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// LoadConfig will load the current sonobuoy configuration using the filesystem
// and environment variables, and returns a config object
func LoadConfig() (*Config, error) {
	var err error
	cfg := New()

	// 0 - load defaults
	viper.SetConfigType("json")
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/sonobuoy/")
	viper.AddConfigPath(".")

	// Allow specifying a custom config file via the SONOBUOY_CONFIG env var
	if forceCfg := os.Getenv("SONOBUOY_CONFIG"); forceCfg != "" {
		viper.SetConfigFile(forceCfg)
	}

	// 1 - Read in the config file.
	if err = viper.ReadInConfig(); err != nil {
		return nil, errors.WithStack(err)
	}

	// 2 - Unmarshal the Config struct
	if err = viper.Unmarshal(cfg); err != nil {
		return nil, errors.WithStack(err)
	}

	// 3 - figure out what address we will tell pods to dial for aggregation
	if cfg.Aggregation.AdvertiseAddress == "" {
		if ip := os.Getenv("SONOBUOY_ADVERTISE_IP"); ip != "" {
			cfg.Aggregation.AdvertiseAddress = fmt.Sprintf("%v:%d", ip, cfg.Aggregation.BindPort)
		} else {
			hostname, _ := os.Hostname()
			if hostname != "" {
				cfg.Aggregation.AdvertiseAddress = fmt.Sprintf("%v:%d", hostname, cfg.Aggregation.BindPort)
			}
		}
	}

	// 4 - Any other settings
	cfg.Version = buildinfo.Version

	// Make the results dir overridable with an environment variable
	if resultsDir, ok := os.LookupEnv("RESULTS_DIR"); ok {
		cfg.ResultsDir = resultsDir
	}

	// Use the exact user config for resources, if set. Viper merges in
	// arrays, making this part necessary.  This way, if they leave out the
	// Resources section altogether they get the default set, but if they
	// set it at all (including to an empty array), we use exactly what
	// they specify.
	if viper.IsSet("Resources") {
		cfg.Resources = viper.GetStringSlice("Resources")
	}

	// 5 - Load any plugins we have
	err = loadAllPlugins(cfg)
	if err != nil {
		return nil, err
	}

	// 6 - Return any validation errors
	validationErrs := cfg.Validate()
	if len(validationErrs) > 0 {
		errstrs := make([]string, len(validationErrs))
		for i := range validationErrs {
			errstrs[i] = validationErrs[i].Error()
		}

		return nil, errors.Errorf("invalid configuration: %v", strings.Join(errstrs, ", "))
	}

	return cfg, err
}

// Validate returns a list of errors for the configuration, if any are found.
func (cfg *Config) Validate() (errors []error) {
	if _, defaulted, err := cfg.Limits.PodLogs.sizeLimitBytes(); err != nil && !defaulted {
		errors = append(errors, err)
	}

	if _, defaulted, err := cfg.Limits.PodLogs.timeLimitDuration(); err != nil && !defaulted {
		errors = append(errors, err)
	}

	return errors
}

// loadAllPlugins takes the given sonobuoy configuration and gives back a
// plugin.Interface for every plugin specified by the configuration.
func loadAllPlugins(cfg *Config) error {
	var plugins []plugin.Interface

	// Load all Plugins
	plugins, err := pluginloader.LoadAllPlugins(
		cfg.Namespace,
		cfg.WorkerImage,
		cfg.ImagePullPolicy,
		cfg.PluginSearchPath,
		cfg.PluginSelections,
	)
	if err != nil {
		return err
	}

	// Find any selected plugins that weren't loaded
	for _, sel := range cfg.PluginSelections {
		found := false
		for _, p := range plugins {
			if p.GetName() == sel.Name {
				found = true
			}
		}

		if !found {
			return errors.Errorf("Configured plugin %v does not exist", sel.Name)
		}
	}

	for _, p := range plugins {
		cfg.addPlugin(p)
	}

	return nil
}
