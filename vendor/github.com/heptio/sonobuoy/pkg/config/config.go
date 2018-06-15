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
	"path"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/satori/go.uuid"
)

const (
	// DefaultNamespace is the namespace where the master and plugin workers will run (but not necessarily the pods created by the plugin workers).
	DefaultNamespace = "heptio-sonobuoy"
	// DefaultKubeConformanceImage is the URL of the docker image to run for the kube conformance tests.
	DefaultKubeConformanceImage = "gcr.io/heptio-images/kube-conformance:latest"
	// DefaultAggregationServerBindPort is the default port for the aggregation server to bind to.
	DefaultAggregationServerBindPort = 8080
	// DefaultAggregationServerBindAddress is the default address for the aggregation server to bind to.
	DefaultAggregationServerBindAddress = "0.0.0.0"
	// MasterPodName is the name of the main pod that runs plugins and collects results.
	MasterPodName = "sonobuoy"
	// MasterContainerName is the name of the main container in the master pod.
	MasterContainerName = "kube-sonobuoy"
	// MasterResultsPath is the location in the main container of the master pod where results will be archived.
	MasterResultsPath = "/tmp/sonobuoy"
)

// DefaultImage is the URL of the docker image to run for the aggregator and workers
var DefaultImage = "gcr.io/heptio-images/sonobuoy:" + buildinfo.Version

///////////////////////////////////////////////////////
// Note: The described resources are a 1:1 match
// with kubectl UX for consistent user experience.
// xref: https://kubernetes.io/docs/api-reference/v1.8/
///////////////////////////////////////////////////////

// ClusterResources is the list of API resources that are scoped to the entire
// cluster (ie. not to any particular namespace)
var ClusterResources = []string{
	"CertificateSigningRequests",
	"ClusterRoleBindings",
	"ClusterRoles",
	"ComponentStatuses",
	"CustomResourceDefinitions",
	"Nodes",
	"PersistentVolumes",
	"PodSecurityPolicies",
	"ServerGroups",
	"ServerVersion",
	"StorageClasses",
	"ThirdPartyResources",
}

// NamespacedResources is the list of API resources that are scoped to a
// kubernetes namespace.
var NamespacedResources = []string{
	"ConfigMaps",
	"ControllerRevisions",
	"CronJobs",
	"DaemonSets",
	"Deployments",
	"Endpoints",
	"Events",
	"HorizontalPodAutoscalers",
	"Ingresses",
	"Jobs",
	"LimitRanges",
	"NetworkPolicies",
	"PersistentVolumeClaims",
	"PodDisruptionBudgets",
	"PodLogs",
	"PodPresets",
	"PodTemplates",
	"Pods",
	"ReplicaSets",
	"ReplicationControllers",
	"ResourceQuotas",
	"RoleBindings",
	"Roles",
	"Secrets",
	"ServiceAccounts",
	"Services",
	"StatefulSets",
}

// FilterOptions allow operators to select sets to include in a report
type FilterOptions struct {
	Namespaces    string `json:"Namespaces"`
	LabelSelector string `json:"LabelSelector"`
}

// Config is the input struct used to determine what data to collect.
type Config struct {
	// NOTE: viper uses "mapstructure" as the tag for config
	// serialization, *NOT* "json".  mapstructure is a separate library
	// that converts maps to structs, and has its own syntax for tagging
	// fields. The only documentation on this is in the mapstructure docs:
	// https://godoc.org/github.com/mitchellh/mapstructure#example-Decode--Tags
	//
	// To be safe we annotate with both json and mapstructure tags.

	///////////////////////////////////////////////
	// Meta-Data collection options
	///////////////////////////////////////////////
	Description string `json:"Description" mapstructure:"Description"`
	UUID        string `json:"UUID" mapstructure:"UUID"`
	Version     string `json:"Version" mapstructure:"Version"`
	ResultsDir  string `json:"ResultsDir" mapstructure:"ResultsDir"`

	///////////////////////////////////////////////
	// Data collection options
	///////////////////////////////////////////////
	Resources []string `json:"Resources" mapstructure:"Resources"`

	///////////////////////////////////////////////
	// Filtering options
	///////////////////////////////////////////////
	Filters FilterOptions `json:"Filters" mapstructure:"Filters"`

	///////////////////////////////////////////////
	// Limit options
	///////////////////////////////////////////////
	Limits LimitConfig `json:"Limits" mapstructure:"Limits"`

	///////////////////////////////////////////////
	// plugin configurations settings
	///////////////////////////////////////////////
	Aggregation      plugin.AggregationConfig `json:"Server" mapstructure:"Server"`
	PluginSelections []plugin.Selection       `json:"Plugins" mapstructure:"Plugins"`
	PluginSearchPath []string                 `json:"PluginSearchPath" mapstructure:"PluginSearchPath"`
	Namespace        string                   `json:"Namespace" mapstructure:"Namespace"`
	LoadedPlugins    []plugin.Interface       // this is assigned when plugins are loaded.

	///////////////////////////////////////////////
	// sonobuoy configuration
	///////////////////////////////////////////////
	WorkerImage     string `json:"WorkerImage" mapstructure:"WorkerImage"`
	ImagePullPolicy string `json:"ImagePullPolicy" mapstructure:"ImagePullPolicy"`
}

// LimitConfig is a configuration on the limits of sizes of various responses.
type LimitConfig struct {
	PodLogs SizeOrTimeLimitConfig `json:"PodLogs" mapstructure:"PodLogs"`
}

// SizeOrTimeLimitConfig represents configuration that limits the size of
// something either by a total disk size, or by a length of time.
type SizeOrTimeLimitConfig struct {
	LimitSize string `json:"LimitSize" mapstructure:"LimitSize"`
	LimitTime string `json:"LimitTime" mapstructure:"LimitTime"`
}

// FilterResources is a utility function used to parse Resources
func (cfg *Config) FilterResources(filter []string) []string {
	var results []string
	for _, felement := range filter {
		for _, check := range cfg.Resources {
			if felement == check {
				results = append(results, felement)
			}
		}
	}
	return results
}

// OutputDir returns the directory under the ResultsDir containing the
// UUID for this run.
func (cfg *Config) OutputDir() string {
	return path.Join(cfg.ResultsDir, cfg.UUID)
}

// SizeLimitBytes returns how many bytes the configuration is set to limit,
// returning defaultVal if not set.
func (c SizeOrTimeLimitConfig) SizeLimitBytes(defaultVal int64) int64 {
	val, defaulted, err := c.sizeLimitBytes()

	// Ignore error, since we should have already caught it in validation
	if err != nil || defaulted {
		return defaultVal
	}

	return val
}

func (c SizeOrTimeLimitConfig) sizeLimitBytes() (val int64, defaulted bool, err error) {
	str := c.LimitSize
	if str == "" {
		return 0, true, nil
	}

	var bs datasize.ByteSize
	err = bs.UnmarshalText([]byte(str))
	return int64(bs.Bytes()), false, err
}

// TimeLimitDuration returns the duration the configuration is set to limit, returning defaultVal if not set.
func (c SizeOrTimeLimitConfig) TimeLimitDuration(defaultVal time.Duration) time.Duration {
	val, defaulted, err := c.timeLimitDuration()

	// Ignore error, since we should have already caught it in validation
	if err != nil || defaulted {
		return defaultVal
	}

	return val
}

func (c SizeOrTimeLimitConfig) timeLimitDuration() (val time.Duration, defaulted bool, err error) {
	str := c.LimitTime
	if str == "" {
		return 0, true, nil
	}

	val, err = time.ParseDuration(str)
	return val, false, err
}

// New returns a newly-constructed Config object with default values.
func New() *Config {
	var cfg Config
	cfg.UUID = uuid.NewV4().String()
	cfg.Description = "DEFAULT"
	cfg.ResultsDir = "/tmp/sonobuoy"
	cfg.Version = buildinfo.Version

	cfg.Filters.Namespaces = ".*"

	cfg.Resources = ClusterResources
	cfg.Resources = append(cfg.Resources, NamespacedResources...)

	cfg.Namespace = DefaultNamespace

	cfg.Aggregation.BindAddress = DefaultAggregationServerBindAddress
	cfg.Aggregation.BindPort = DefaultAggregationServerBindPort
	cfg.Aggregation.TimeoutSeconds = 5400 // 90 minutes

	cfg.PluginSearchPath = []string{
		"./plugins.d",
		"/etc/sonobuoy/plugins.d",
		"~/sonobuoy/plugins.d",
	}

	// TODO (timothysc) reference the other consts
	cfg.WorkerImage = "gcr.io/heptio-images/sonobuoy:latest"
	cfg.ImagePullPolicy = "Always"

	return &cfg
}

// addPlugin adds a (configured, initialized) plugin to the config object so
// that it can be executed.
func (cfg *Config) addPlugin(plugin plugin.Interface) {
	cfg.LoadedPlugins = append(cfg.LoadedPlugins, plugin)
}

// getPlugins gets the list of plugins selected for this configuration.
func (cfg *Config) getPlugins() []plugin.Interface {
	return cfg.LoadedPlugins
}
