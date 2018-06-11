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
	"bytes"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/heptio/sonobuoy/pkg/templates"
)

// templateValues are used for direct template substitution for manifest generation.
type templateValues struct {
	E2EFocus             string
	E2ESkip              string
	E2EParallel          string
	SonobuoyConfig       string
	SonobuoyImage        string
	Version              string
	Namespace            string
	EnableRBAC           bool
	ImagePullPolicy      string
	KubeConformanceImage string
}

// GenerateManifest fills in a template with a Sonobuoy config
func (c *SonobuoyClient) GenerateManifest(cfg *GenConfig) ([]byte, error) {
	marshalledConfig, err := json.Marshal(cfg.Config)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't marshall selector")
	}

	// Template values that are regexps (`E2EFocus` and `E2ESkip`) are
	// embedded in YAML files using single quotes to remove the need to
	// escape characters e.g. `\` as they would be if using double quotes.
	// As these strings are regexps, it is expected that they will contain,
	// among other characters, backslashes. Only single quotes need to be
	// escaped in single quote YAML strings, hence the substitions below.
	// See http://www.yaml.org/spec/1.2/spec.html#id2788097 for more details
	// on YAML escaping.
	tmplVals := &templateValues{
		E2EFocus:             strings.Replace(cfg.E2EConfig.Focus, "'", "''", -1),
		E2ESkip:              strings.Replace(cfg.E2EConfig.Skip, "'", "''", -1),
		E2EParallel:          strings.Replace(cfg.E2EConfig.Parallel, "'", "''", -1),
		SonobuoyConfig:       string(marshalledConfig),
		SonobuoyImage:        cfg.Image,
		Version:              buildinfo.Version,
		Namespace:            cfg.Namespace,
		EnableRBAC:           cfg.EnableRBAC,
		ImagePullPolicy:      cfg.ImagePullPolicy,
		KubeConformanceImage: cfg.KubeConformanceImage,
	}

	var buf bytes.Buffer

	if err := templates.Manifest.Execute(&buf, tmplVals); err != nil {
		return nil, errors.Wrap(err, "couldn't execute manifest template")
	}

	return buf.Bytes(), nil
}
