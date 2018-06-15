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
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
)

func (c *SonobuoyClient) GetStatus(namespace string) (*aggregation.Status, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}

	if _, err := client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{}); err != nil {
		return nil, errors.Wrap(err, "sonobuoy namespace does not exist")
	}

	pod, err := client.CoreV1().Pods(namespace).Get(aggregation.StatusPodName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve sonobuoy pod")
	}

	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("pod has status %q", pod.Status.Phase)
	}

	statusJSON, ok := pod.Annotations[aggregation.StatusAnnotationName]
	if !ok {
		return nil, fmt.Errorf("missing status annotation %q", aggregation.StatusAnnotationName)
	}

	var status aggregation.Status
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		return nil, errors.Wrap(err, "couldn't unmarshal the JSON status annotation")
	}

	return &status, nil
}
