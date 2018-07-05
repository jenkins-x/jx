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

package aggregation

import "fmt"

const (
	// RunningStatus means the sonobuoy run is still in progress.
	RunningStatus string = "running"
	// CompleteStatus means the sonobuoy run is complete.
	CompleteStatus string = "complete"
	// FailedStatus means one or more plugins has failed and the run will not complete successfully.
	FailedStatus string = "failed"
)

// PluginStatus represents the current status of an individual plugin.
type PluginStatus struct {
	Plugin string `json:"plugin"`
	Node   string `json:"node"`
	Status string `json:"status"`
}

// Status represents the current status of a Sonobuoy run.
// TODO(EKF): Find a better name for this struct/package.
type Status struct {
	Plugins []PluginStatus `json:"plugins"`
	Status  string         `json:"status"`
}

func (s *Status) updateStatus() error {
	status := CompleteStatus
	for _, plugin := range s.Plugins {
		switch plugin.Status {
		case CompleteStatus:
			continue
		case FailedStatus:
			status = FailedStatus
		case RunningStatus:
			if status != FailedStatus {
				status = RunningStatus
			}
		default:
			return fmt.Errorf("unknown status %s", plugin.Status)
		}
	}
	s.Status = status
	return nil
}
