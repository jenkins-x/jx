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
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
)

// QueryRecorder records a sequence of queries
type QueryRecorder struct {
	queries []*QueryData
}

// NewQueryRecorder returns a new empty QueryRecorder
func NewQueryRecorder() *QueryRecorder {
	return &QueryRecorder{
		queries: make([]*QueryData, 0),
	}
}

// QueryData captures the results of the run for post-processing
type QueryData struct {
	QueryObj    string `json:"queryobj,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	ElapsedTime string `json:"time,omitempty"`
	Error       error  `json:"error,omitempty"`
}

// RecordQuery transcribes a query by name, namespace, duration and error
func (q *QueryRecorder) RecordQuery(name string, namespace string, duration time.Duration, recerr error) {
	if recerr != nil {
		errlog.LogError(errors.Wrapf(recerr, "error querying %v", name))
	}
	summary := &QueryData{
		QueryObj:    name,
		Namespace:   namespace,
		ElapsedTime: duration.String(),
		Error:       recerr,
	}

	q.queries = append(q.queries, summary)
}

// DumpQueryData writes query information out to a file at the give filepath
func (q *QueryRecorder) DumpQueryData(filepath string) error {
	// Ensure the leading path is created
	err := os.MkdirAll(path.Dir(filepath), 0755)
	if err != nil {
		return err
	}

	// Format the query data as JSON
	data, err := json.Marshal(q.queries)
	if err != nil {
		return err
	}

	// Create the file
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write the data
	_, err = f.Write(data)
	return err
}
