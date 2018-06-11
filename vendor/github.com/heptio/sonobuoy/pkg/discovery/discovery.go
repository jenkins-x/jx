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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/errlog"
	pluginaggregation "github.com/heptio/sonobuoy/pkg/plugin/aggregation"
	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/viniciuschiele/tarx"
	"k8s.io/client-go/kubernetes"
)

// Run is the main entrypoint for discovery
func Run(kubeClient kubernetes.Interface, cfg *config.Config) (errCount int) {
	t := time.Now()

	// 1. Create the directory which will store the results, including the
	// `meta` directory inside it (which we always need regardless of
	// config)
	outpath := path.Join(cfg.ResultsDir, cfg.UUID)
	metapath := path.Join(outpath, MetaLocation)
	err := os.MkdirAll(metapath, 0755)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create directory to store results"))
		return errCount + 1
	}

	// Write logs to the configured results location. All log levels
	// should write to the same log file
	pathmap := make(lfshook.PathMap)
	logfile := path.Join(metapath, "run.log")
	for _, level := range logrus.AllLevels {
		pathmap[level] = logfile
	}

	hook := lfshook.NewHook(pathmap, &logrus.JSONFormatter{})

	logrus.AddHook(hook)

	// Unset all hooks as we exit the Run function
	defer func() {
		logrus.StandardLogger().Hooks = make(logrus.LevelHooks)
	}()
	// closure used to collect and report errors.
	trackErrorsFor := func(action string) func(error) {
		return func(err error) {
			if err != nil {
				errCount++
				errlog.LogError(errors.Wrapf(err, "error %v", action))
			}
		}
	}

	// 2. Get the list of namespaces and apply the regex filter on the namespace
	nsfilter := fmt.Sprintf("%s|%s", cfg.Filters.Namespaces, cfg.Namespace)
	logrus.Infof("Filtering namespaces based on the following regex:%s", nsfilter)
	nslist, err := FilterNamespaces(kubeClient, nsfilter)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not filter namespaces"))
		return errCount + 1
	}

	// 3. Dump the config.json we used to run our test
	if blob, err := json.Marshal(cfg); err == nil {
		if err = ioutil.WriteFile(path.Join(metapath, "config.json"), blob, 0644); err != nil {
			errlog.LogError(errors.Wrap(err, "could not write config.json file"))
			return errCount + 1
		}
	}

	// 4. Run the plugin aggregator
	trackErrorsFor("running plugins")(
		pluginaggregation.Run(kubeClient, cfg.LoadedPlugins, cfg.Aggregation, cfg.Namespace, outpath),
	)

	// 5. Run the queries
	recorder := NewQueryRecorder()
	trackErrorsFor("querying cluster resources")(
		QueryClusterResources(kubeClient, recorder, cfg),
	)

	for _, ns := range nslist {
		trackErrorsFor("querying resources under namespace " + ns)(
			QueryNSResources(kubeClient, recorder, ns, cfg),
		)
	}

	// 6. Dump the query times
	trackErrorsFor("recording query times")(
		recorder.DumpQueryData(path.Join(metapath, "query-time.json")),
	)

	// 7. Clean up after the plugins
	pluginaggregation.Cleanup(kubeClient, cfg.LoadedPlugins)

	// 8. tarball up results YYYYMMDDHHMM_sonobuoy_UID.tar.gz
	tb := cfg.ResultsDir + "/" + t.Format("200601021504") + "_sonobuoy_" + cfg.UUID + ".tar.gz"
	err = tarx.Compress(tb, outpath, &tarx.CompressOptions{Compression: tarx.Gzip})
	if err == nil {
		defer os.RemoveAll(outpath)
	}
	trackErrorsFor("assembling results tarball")(err)
	logrus.Infof("Results available at %v", tb)

	return errCount
}
