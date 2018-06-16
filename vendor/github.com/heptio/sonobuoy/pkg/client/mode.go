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
	"fmt"
	"strings"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

// Mode identifies a specific mode of running Sonobuoy.
// A mode is a defined configuration of plugins and E2E Focus and Config.
// Modes form the base level defaults, which can then be overriden by the e2e flags
// and the config flag.
type Mode string

const (
	// Quick runs a single E2E test and the systemd log tests.
	Quick Mode = "Quick"
	// Conformance runs all of the E2E tests and the systemd log tests.
	Conformance Mode = "Conformance"
	// Extended run all of the E2E tests, the systemd log tests, and
	// Heptio's E2E Tests.
	Extended Mode = "Extended"
)

const defaultSkipList = `Alpha|Kubectl|\[(Disruptive|Feature:[^\]]+|Flaky)\]`

var modeMap = map[string]Mode{
	string(Conformance): Conformance,
	string(Quick):       Quick,
	string(Extended):    Extended,
}

// ModeConfig represents the sonobuoy configuration for a given mode.
type ModeConfig struct {
	// E2EConfig is the focus and skip vars for the conformance tests.
	E2EConfig E2EConfig
	// Selectors are the plugins selected by this mode.
	Selectors []plugin.Selection
}

// String needed for pflag.Value
func (m *Mode) String() string { return string(*m) }

// Type needed for pflag.Value
func (m *Mode) Type() string { return "Mode" }

// Set the name with a given string. Returns error on unknown mode.
func (m *Mode) Set(str string) error {
	// Allow lowercase "conformance", "quick" etc in command line
	upcase := strings.Title(str)
	mode, ok := modeMap[upcase]
	if !ok {
		return fmt.Errorf("unknown mode %s", str)
	}
	*m = mode
	return nil
}

// Get returns the ModeConfig associated with a mode name, or nil
// if there's no associated mode
func (m *Mode) Get() *ModeConfig {
	switch *m {
	case Conformance:
		return &ModeConfig{
			E2EConfig: E2EConfig{
				Focus:    `\[Conformance\]`,
				Skip:     defaultSkipList,
				Parallel: "1",
			},
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
			},
		}
	case Quick:
		return &ModeConfig{
			E2EConfig: E2EConfig{
				Focus:    "Pods should be submitted and removed",
				Skip:     defaultSkipList,
				Parallel: "1",
			},
			Selectors: []plugin.Selection{
				{Name: "e2e"},
			},
		}
	case Extended:
		return &ModeConfig{
			E2EConfig: E2EConfig{
				Focus:    `\[Conformance\]`,
				Skip:     defaultSkipList,
				Parallel: "1",
			},
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
				{Name: "heptio-e2e"},
			},
		}
	default:
		return nil
	}
}

// GetModes gets a list of all available modes.
func GetModes() []string {
	keys := make([]string, len(modeMap))
	i := 0
	for k := range modeMap {
		keys[i] = k
		i++
	}
	return keys
}
