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
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	clusterRoleFieldName  = "component"
	clusterRoleFieldValue = "sonobuoy"

	e2eNamespacePrefix = "e2e-"
)

func (c *SonobuoyClient) Delete(cfg *DeleteConfig) error {
	client, err := c.Client()
	if err != nil {
		return err
	}

	if err := cleanupNamespace(cfg.Namespace, client); err != nil {
		return err
	}

	if cfg.EnableRBAC {
		if err := deleteRBAC(client); err != nil {
			return err
		}
	}

	if cfg.DeleteAll {
		if err := cleanupE2E(client); err != nil {
			return err
		}
	}
	return nil
}

func cleanupNamespace(namespace string, client kubernetes.Interface) error {
	// Delete the namespace
	log := logrus.WithFields(logrus.Fields{
		"kind":      "namespace",
		"namespace": namespace,
	})

	err := client.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{})
	if err := logDelete(log, err); err != nil {
		return errors.Wrap(err, "couldn't delete namespace")
	}

	return nil
}

func deleteRBAC(client kubernetes.Interface) error {
	// ClusterRole and ClusterRoleBindings aren't namespaced, so delete them seperately
	selector := metav1.AddLabelToSelector(
		&metav1.LabelSelector{},
		clusterRoleFieldName,
		clusterRoleFieldValue,
	)

	deleteOpts := &metav1.DeleteOptions{}
	listOpts := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(selector),
	}

	err := client.RbacV1().ClusterRoleBindings().DeleteCollection(deleteOpts, listOpts)
	if err := logDelete(logrus.WithField("kind", "clusterrolebindings"), err); err != nil {
		return errors.Wrap(err, "failed to delete cluster role binding")
	}

	// ClusterRole and ClusterRole bindings aren't namespaced, so delete them manually
	err = client.RbacV1().ClusterRoles().DeleteCollection(deleteOpts, listOpts)
	if err := logDelete(logrus.WithField("kind", "clusterroles"), err); err != nil {
		return errors.Wrap(err, "failed to delete cluster role")
	}

	return nil
}

func cleanupE2E(client kubernetes.Interface) error {
	// Delete any dangling E2E namespaces

	namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list namespaces")
	}

	for _, namespace := range namespaces.Items {
		if strings.HasPrefix(namespace.Name, e2eNamespacePrefix) {

			log := logrus.WithFields(logrus.Fields{
				"kind":      "namespace",
				"namespace": namespace.Name,
			})
			err := client.CoreV1().Namespaces().Delete(namespace.Name, &metav1.DeleteOptions{})
			if err := logDelete(log, err); err != nil {
				return errors.Wrap(err, "couldn't delete namespace")
			}
		}
	}
	return nil
}

func logDelete(log logrus.FieldLogger, err error) error {
	switch {
	case err == nil:
		log.Info("deleted")
	case kubeerror.IsNotFound(err):
		log.Info("already deleted")
	case kubeerror.IsConflict(err):
		log.WithError(err).Info("delete in progress")
	case err != nil:
		return err
	}
	return nil
}
