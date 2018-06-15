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

	version "github.com/hashicorp/go-version"
	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var preflightChecks = []func(kubernetes.Interface, *PreflightConfig) error{
	preflightDNSCheck,
	preflightVersionCheck,
	preflightExistingSonobuoy,
}

// PreflightChecks runs all preflight checks in order, returning the first error encountered.
func (c *SonobuoyClient) PreflightChecks(cfg *PreflightConfig) []error {
	client, err := c.Client()
	if err != nil {
		return []error{err}
	}

	errors := []error{}

	for _, check := range preflightChecks {
		if err := check(client, cfg); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

const (
	kubeSystemNamespace = "kube-system"
	kubeDNSLabelKey     = "k8s-app"
	kubeDNSLabelValue   = "kube-dns"
	coreDNSLabelValue   = "coredns"
)

func preflightDNSCheck(client kubernetes.Interface, cfg *PreflightConfig) error {
	var dnsLabels = []string{
		kubeDNSLabelValue,
		coreDNSLabelValue,
	}

	var nPods = 0
	for _, labelValue := range dnsLabels {
		selector := metav1.AddLabelToSelector(&metav1.LabelSelector{}, kubeDNSLabelKey, labelValue)

		obj, err := client.CoreV1().Pods(kubeSystemNamespace).List(
			metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)},
		)
		if err != nil {
			return errors.Wrap(err, "could not retrieve list of pods")
		}

		nPods += len(obj.Items)
	}

	if nPods == 0 {
		return errors.New("no dns pod tests found")
	}

	return nil
}

var (
	minimumKubeVersion = version.Must(version.NewVersion(buildinfo.MinimumKubeVersion))
	maximumKubeVersion = version.Must(version.NewVersion(buildinfo.MaximumKubeVersion))
)

func preflightVersionCheck(client kubernetes.Interface, cfg *PreflightConfig) error {
	versionInfo, err := client.Discovery().ServerVersion()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve server version")
	}

	serverVersion, err := version.NewVersion(versionInfo.String())
	if err != nil {
		return errors.Wrap(err, "couldn't parse version string")
	}

	if serverVersion.LessThan(minimumKubeVersion) {
		return fmt.Errorf("Minimum kubernetes version is %s, got %s", minimumKubeVersion.String(), versionInfo.String())
	}

	if serverVersion.GreaterThan(maximumKubeVersion) {
		return fmt.Errorf("Maximum kubernetes version is %s, got %s", maximumKubeVersion.String(), versionInfo.String())
	}

	return nil
}

func preflightExistingSonobuoy(client kubernetes.Interface, cfg *PreflightConfig) error {
	_, err := client.CoreV1().Pods(cfg.Namespace).Get("sonobuoy", metav1.GetOptions{})
	switch {
	// Pod doesn't exist: great!
	case apierrors.IsNotFound(err):
		return nil
	case err != nil:
		return errors.Wrap(err, "error checking for Sonobuoy pod")
	// No error: pod exists
	case err == nil:
		return errors.New("sonobuoy run already exists in this namespace")
	}
	return nil
}
