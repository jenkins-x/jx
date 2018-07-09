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
	"io"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

const bufferSize = 4096

func (c *SonobuoyClient) Run(cfg *RunConfig) error {
	manifest, err := c.GenerateManifest(&cfg.GenConfig)
	if err != nil {
		return errors.Wrap(err, "couldn't run invalid manifest")
	}

	buf := bytes.NewBuffer(manifest)
	d := yaml.NewYAMLOrJSONDecoder(buf, bufferSize)

	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "couldn't decode template")
		}

		// Skip over empty or partial objects
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}

		obj := &unstructured.Unstructured{}
		if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), ext.Raw, obj); err != nil {
			return errors.Wrap(err, "couldn't decode template")
		}
		name, err := c.dynamicClient.Name(obj)
		if err != nil {
			return errors.Wrap(err, "could not get object name")
		}
		namespace, err := c.dynamicClient.Namespace(obj)
		if err != nil {
			return errors.Wrap(err, "could not get object namespace")
		}
		// err is used to determine output for user; but first extract resource
		_, err = c.dynamicClient.CreateObject(obj)
		resource, err2 := c.dynamicClient.ResourceVersion(obj)
		if err2 != nil {
			return errors.Wrap(err, "could not get resource of object")
		}
		if err := handleCreateError(name, namespace, resource, err); err != nil {
			return errors.Wrap(err, "failed to create object")
		}
	}
	return nil
}

func handleCreateError(name, namespace, resource string, err error) error {
	log := logrus.WithFields(logrus.Fields{
		"name":      name,
		"namespace": namespace,
		"resource":  resource,
	})

	switch {
	case err == nil:
		log.Info("created object")
	// Some resources (like ClusterRoleBinding and ClusterBinding) aren't
	// namespaced and may overlap between runs. So don't abort on duplicate errors
	// in this case.
	case namespace == "" && kubeerror.IsAlreadyExists(err):
		log.Info("object already exists")
	case err != nil:
		return errors.Wrapf(err, "failed to create API resource %s", name)
	}
	return nil
}
