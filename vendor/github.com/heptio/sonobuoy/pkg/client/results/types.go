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

package results

import (
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/discovery"
)

// Metadata is the data about the Sonobuoy run and how long it took to query the
// system.
type Metadata struct {
	// Config is the config used during this Sonobuoy run.
	Config config.Config

	// QueryMetadata shows information about each query Sonobuoy ran in the
	// cluster.
	QueryData []discovery.QueryData
}
