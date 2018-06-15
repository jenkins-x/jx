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
	"io/ioutil"
	"os"
	"regexp"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// FilterNamespaces filter the list of namespaces according to the filter string
func FilterNamespaces(kubeClient kubernetes.Interface, filter string) ([]string, error) {
	var validns []string
	re := regexp.MustCompile(filter)
	nslist, err := kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, ns := range nslist.Items {
		logrus.Infof("Namespace %v Matched=%v", ns.Name, re.MatchString(ns.Name))
		if re.MatchString(ns.Name) {
			validns = append(validns, ns.Name)
		}
	}
	return validns, nil
}

// SerializeObj will write out an object
func SerializeObj(obj interface{}, outpath string, file string) error {
	var err error
	if err = os.MkdirAll(outpath, 0755); err == nil {
		if eJSONBytes, err := json.Marshal(obj); err == nil {
			err = ioutil.WriteFile(outpath+"/"+file, eJSONBytes, 0644)
		}
	}
	return errors.WithStack(err)
}

// SerializeArrayObj will write out an array of object
func SerializeArrayObj(objs []interface{}, outpath string, file string) error {
	var err error
	if err = os.MkdirAll(outpath, 0755); err == nil {
		if eJSONBytes, err := json.Marshal(objs); err == nil {
			err = ioutil.WriteFile(outpath+"/"+file, eJSONBytes, 0644)
		}
	}
	return errors.WithStack(err)
}

// SerializeObjAppend will serialize an object and append to the end of file
func SerializeObjAppend(f *os.File, obj interface{}) error {
	var err error
	if blob, err := json.Marshal(obj); err == nil {
		_, err = f.Write(blob)
		_, err = f.WriteString(",")
	}
	return errors.WithStack(err)
}
