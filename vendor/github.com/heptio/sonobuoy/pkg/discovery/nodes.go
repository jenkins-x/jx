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

package discovery

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type nodeData struct {
	APIResource   v1.Node                `json:"apiResource,omitempty"`
	ConfigzOutput map[string]interface{} `json:"configzOutput,omitempty"`
	HealthzStatus int                    `json:"healthzStatus,omitempty"`
}

// getNodeEndpoint returns the response from pinging a node endpoint
func getNodeEndpoint(client rest.Interface, nodeName, endpoint string) (rest.Result, error) {
	// TODO(chuckha) make this timeout configurable
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30*time.Second))
	defer cancel()
	req := client.
		Get().
		Context(ctx).
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy").
		Suffix(endpoint)

	result := req.Do()
	if result.Error() != nil {
		logrus.Warningf("Could not get %v endpoint for node %v: %v", endpoint, nodeName, result.Error())
	}
	return result, result.Error()
}

// gatherNodeData collects non-resource information about a node through the
// kubernetes API.  That is, its `healthz` and `configz` endpoints, which are
// not "resources" per se, although they are accessible through the apiserver.
func gatherNodeData(nodeNames []string, restclient rest.Interface, cfg *config.Config) error {
	logrus.Info("Collecting Node Configuration and Health...")

	for _, name := range nodeNames {
		// Create the output for each node
		out := path.Join(cfg.OutputDir(), HostsLocation, name)
		logrus.Infof("Creating host results for %v under %v\n", name, out)
		if err := os.MkdirAll(out, 0755); err != nil {
			return err
		}

		_, err := untypedQuery(out, "configz.json", func() (interface{}, error) {
			data := make(map[string]interface{})
			result, err := getNodeEndpoint(restclient, name, "configz")
			if err != nil {
				return data, err
			}

			resultBytes, err := result.Raw()
			if err != nil {
				return data, err
			}
			json.Unmarshal(resultBytes, &data)
			return data, err
		})
		if err != nil {
			return err
		}

		_, err = untypedQuery(out, "healthz.json", func() (interface{}, error) {
			data := make(map[string]interface{})
			result, err := getNodeEndpoint(restclient, name, "healthz")
			if err != nil {
				return data, err
			}
			var healthstatus int
			result.StatusCode(&healthstatus)
			data["status"] = healthstatus
			return data, nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
