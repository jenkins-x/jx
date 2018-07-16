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
	"os"
	"path"
	"time"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// ObjQuery is a query function that returns a kubernetes object
type ObjQuery func() (runtime.Object, error)

// UntypedQuery is a query function that return an untyped array of objs
type UntypedQuery func() (interface{}, error)

// UntypedListQuery is a query function that return an untyped array of objs
type UntypedListQuery func() ([]interface{}, error)

const (
	// NSResourceLocation is the place under which namespaced API resources (pods, etc) are stored
	NSResourceLocation = "resources/ns"
	// ClusterResourceLocation is the place under which non-namespaced API resources (nodes, etc) are stored
	ClusterResourceLocation = "resources/cluster"
	// HostsLocation is the place under which host information (configz, healthz) is stored
	HostsLocation = "hosts"
	// MetaLocation is the place under which snapshot metadata (query times, config) is stored
	MetaLocation = "meta"
)

// objListQuery performs a list query and serialize the results
func objListQuery(outpath string, file string, f ObjQuery) (time.Duration, error) {
	start := time.Now()
	listObj, err := f()
	duration := time.Since(start)
	if err != nil {
		return duration, err
	}
	if listObj == nil {
		return duration, errors.Errorf("got invalid response from API server")
	}

	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return duration, errors.WithStack(err)
	}

	items, err := conversion.EnforcePtr(listPtr)
	if err != nil {
		return duration, errors.WithStack(err)
	}

	if items.Len() > 0 {
		err = errors.WithStack(SerializeObj(listPtr, outpath, file))
	}
	return duration, err
}

// untypedQuery performs a untyped query and serialize the results
func untypedQuery(outpath string, file string, f UntypedQuery) (time.Duration, error) {
	start := time.Now()
	Obj, err := f()
	duration := time.Since(start)
	if err == nil && Obj != nil {
		err = SerializeObj(Obj, outpath, file)
	}
	return duration, err
}

// untypedListQuery performs a untyped list query and serialize the results
func untypedListQuery(outpath string, file string, f UntypedListQuery) (time.Duration, error) {
	start := time.Now()
	listObj, err := f()
	duration := time.Since(start)
	if err == nil && listObj != nil {
		err = SerializeArrayObj(listObj, outpath, file)
	}
	return duration, err
}

// timedQuery Wraps the execution of the function with a recorded timed snapshot
func timedQuery(recorder *QueryRecorder, name string, ns string, fn func() (time.Duration, error)) {
	duration, fnErr := fn()
	recorder.RecordQuery(name, ns, duration, fnErr)
}

// queryNsResource performs the appropriate namespace-scoped query according to its input args
func queryNsResource(ns string, resourceKind string, opts metav1.ListOptions, kubeClient kubernetes.Interface) (runtime.Object, error) {
	switch resourceKind {
	case "ConfigMaps":
		return kubeClient.CoreV1().ConfigMaps(ns).List(opts)
	case "ControllerRevisions":
		lst, err := kubeClient.AppsV1().ControllerRevisions(ns).List(opts)
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.AppsV1beta2().ControllerRevisions(ns).List(opts)
			if apierrors.IsNotFound(err) {
				return kubeClient.AppsV1beta1().ControllerRevisions(ns).List(opts)
			}
			return lst, err
		}
		return lst, err
	case "CronJobs":
		lst, err := kubeClient.BatchV1beta1().CronJobs(ns).List(opts)
		if apierrors.IsNotFound(err) {
			return kubeClient.BatchV2alpha1().CronJobs(ns).List(opts)
		}
		return lst, err
	case "DaemonSets":
		lst, err := kubeClient.AppsV1().DaemonSets(ns).List(opts)
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.AppsV1beta2().DaemonSets(ns).List(opts)
			if apierrors.IsNotFound(err) {
				return kubeClient.ExtensionsV1beta1().DaemonSets(ns).List(opts)
			}
			return lst, err
		}
		return lst, err
	case "Deployments":
		lst, err := kubeClient.AppsV1().Deployments(ns).List(opts)
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.AppsV1beta2().Deployments(ns).List(opts)
			if apierrors.IsNotFound(err) {
				return kubeClient.AppsV1beta1().Deployments(ns).List(opts)
			}
			return lst, err
		}
		return lst, err
	case "Endpoints":
		return kubeClient.CoreV1().Endpoints(ns).List(opts)
	case "Events":
		return kubeClient.CoreV1().Events(ns).List(opts)
	case "HorizontalPodAutoscalers":
		return kubeClient.AutoscalingV1().HorizontalPodAutoscalers(ns).List(opts)
	case "Ingresses":
		return kubeClient.ExtensionsV1beta1().Ingresses(ns).List(opts)
	case "Jobs":
		return kubeClient.BatchV1().Jobs(ns).List(opts)
	case "LimitRanges":
		return kubeClient.CoreV1().LimitRanges(ns).List(opts)
	case "NetworkPolicies":
		return kubeClient.NetworkingV1().NetworkPolicies(ns).List(opts)
	case "PersistentVolumeClaims":
		return kubeClient.CoreV1().PersistentVolumeClaims(ns).List(opts)
	case "Pods":
		return kubeClient.CoreV1().Pods(ns).List(opts)
	case "PodDisruptionBudgets":
		return kubeClient.PolicyV1beta1().PodDisruptionBudgets(ns).List(opts)
	case "PodPresets":
		return kubeClient.SettingsV1alpha1().PodPresets(ns).List(opts)
	case "PodTemplates":
		return kubeClient.CoreV1().PodTemplates(ns).List(opts)
	case "ReplicaSets":
		lst, err := kubeClient.AppsV1().ReplicaSets(ns).List(opts)
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.AppsV1beta2().ReplicaSets(ns).List(opts)
			if apierrors.IsNotFound(err) {
				return kubeClient.ExtensionsV1beta1().ReplicaSets(ns).List(opts)
			}
			return lst, err
		}
		return lst, err
	case "ReplicationControllers":
		return kubeClient.CoreV1().ReplicationControllers(ns).List(opts)
	case "ResourceQuotas":
		return kubeClient.CoreV1().ResourceQuotas(ns).List(opts)
	case "RoleBindings":
		lst, err := kubeClient.RbacV1().RoleBindings(ns).List(opts)
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.RbacV1beta1().RoleBindings(ns).List(opts)
			if apierrors.IsNotFound(err) {
				return kubeClient.RbacV1alpha1().RoleBindings(ns).List(opts)
			}
			return lst, err
		}
		return lst, err
	case "Roles":
		lst, err := kubeClient.RbacV1().Roles(ns).List(opts)
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.RbacV1beta1().Roles(ns).List(opts)
			if apierrors.IsNotFound(err) {
				return kubeClient.RbacV1alpha1().Roles(ns).List(opts)
			}
			return lst, err
		}
		return lst, err
	case "Secrets":
		return kubeClient.CoreV1().Secrets(ns).List(opts)
	case "ServiceAccounts":
		return kubeClient.CoreV1().ServiceAccounts(ns).List(opts)
	case "Services":
		return kubeClient.CoreV1().Services(ns).List(opts)
	case "StatefulSets":
		lst, err := kubeClient.AppsV1().StatefulSets(ns).List(opts)
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.AppsV1beta2().StatefulSets(ns).List(opts)
			if apierrors.IsNotFound(err) {
				return kubeClient.AppsV1beta1().StatefulSets(ns).List(opts)
			}
			return lst, err
		}
		return lst, err
	default:
		return nil, errors.Errorf("Unsupported resource %v", resourceKind)
	}
}

// queryNonNsResource performs the appropriate non-namespace-scoped query according to its input args
func queryNonNsResource(resourceKind string, kubeClient kubernetes.Interface) (runtime.Object, error) {
	switch resourceKind {
	case "CertificateSigningRequests":
		return kubeClient.CertificatesV1beta1().CertificateSigningRequests().List(metav1.ListOptions{})
	case "ClusterRoleBindings":
		lst, err := kubeClient.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.RbacV1beta1().ClusterRoleBindings().List(metav1.ListOptions{})
			if apierrors.IsNotFound(err) {
				return kubeClient.RbacV1alpha1().ClusterRoleBindings().List(metav1.ListOptions{})
			}
			return lst, err
		}
		return lst, err
	case "ClusterRoles":
		lst, err := kubeClient.RbacV1().ClusterRoles().List(metav1.ListOptions{})
		if apierrors.IsNotFound(err) {
			lst, err := kubeClient.RbacV1beta1().ClusterRoles().List(metav1.ListOptions{})
			if apierrors.IsNotFound(err) {
				return kubeClient.RbacV1alpha1().ClusterRoles().List(metav1.ListOptions{})
			}
			return lst, err
		}
		return lst, err
	case "ComponentStatuses":
		return kubeClient.CoreV1().ComponentStatuses().List(metav1.ListOptions{})
	case "CustomResourceDefinitions":
		// TODO : This should get cleaned up in future revisions.
		apiextensionsclientset := apiextensionsclient.New(kubeClient.CoreV1().RESTClient())
		if apiextensionsclientset != nil {
			return apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().List(metav1.ListOptions{})
		}
		return nil, errors.Errorf("Failed to create extensions client for CRDs")
	case "Nodes":
		return kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	case "PersistentVolumes":
		return kubeClient.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	case "PodSecurityPolicies":
		return kubeClient.ExtensionsV1beta1().PodSecurityPolicies().List(metav1.ListOptions{})
	case "StorageClasses":
		lst, err := kubeClient.StorageV1().StorageClasses().List(metav1.ListOptions{})
		if apierrors.IsNotFound(err) {
			return kubeClient.StorageV1beta1().StorageClasses().List(metav1.ListOptions{})
		}
		return lst, err
	default:
		return nil, errors.Errorf("don't know how to handle non-namespaced resource %v", resourceKind)
	}
}

// QueryNSResources will query namespace-specific resources in the cluster,
// writing them out to <resultsdir>/resources/ns/<ns>/*.json
// TODO: Eliminate dependencies from config.Config and pass in data
func QueryNSResources(kubeClient kubernetes.Interface, recorder *QueryRecorder, ns string, cfg *config.Config) error {
	logrus.Infof("Running ns query (%v)", ns)

	// 1. Create the parent directory we will use to store the results
	outdir := path.Join(cfg.OutputDir(), NSResourceLocation, ns)
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return errors.WithStack(err)
	}

	// 2. Setup label filter if there is one.
	opts := metav1.ListOptions{}
	if len(cfg.Filters.LabelSelector) > 0 {
		if _, err := labels.Parse(cfg.Filters.LabelSelector); err != nil {
			logrus.Warningf("Labelselector %v failed to parse with error %v", cfg.Filters.LabelSelector, err)
		} else {
			opts.LabelSelector = cfg.Filters.LabelSelector
		}
	}

	var resources []string
	if cfg.Namespace == ns {
		resources = config.NamespacedResources
	} else {
		resources = cfg.FilterResources(config.NamespacedResources)
	}

	// 3. Execute the ns-query
	for _, resourceKind := range resources {
		switch resourceKind {
		case "PodLogs":
			start := time.Now()
			err := gatherPodLogs(kubeClient, ns, opts, cfg)
			if err != nil {
				return err
			}
			duration := time.Since(start)
			recorder.RecordQuery("PodLogs", ns, duration, err)
		default:
			lister := func() (runtime.Object, error) { return queryNsResource(ns, resourceKind, opts, kubeClient) }
			query := func() (time.Duration, error) { return objListQuery(outdir+"/", resourceKind+".json", lister) }
			timedQuery(recorder, resourceKind, ns, query)
		}
	}

	return nil
}

// QueryClusterResources queries non-namespace resources in the cluster, writing
// them out to <resultsdir>/resources/non-ns/*.json
// TODO: Eliminate dependencies from config.Config and pass in data
func QueryClusterResources(kubeClient kubernetes.Interface, recorder *QueryRecorder, cfg *config.Config) error {
	logrus.Infof("Running non-ns query")

	resources := cfg.FilterResources(config.ClusterResources)

	// 1. Create the parent directory we will use to store the results
	outdir := path.Join(cfg.OutputDir(), ClusterResourceLocation)
	if len(resources) > 0 {
		if err := os.MkdirAll(outdir, 0755); err != nil {
			return errors.WithStack(err)
		}
	}

	// 2. Execute the non-ns-query
	for _, resourceKind := range resources {
		switch resourceKind {
		case "ServerVersion":
			objqry := func() (interface{}, error) { return kubeClient.Discovery().ServerVersion() }
			query := func() (time.Duration, error) {
				return untypedQuery(cfg.OutputDir(), "serverversion.json", objqry)
			}
			timedQuery(recorder, "serverversion", "", query)
		case "ServerGroups":
			objqry := func() (interface{}, error) { return kubeClient.Discovery().ServerGroups() }
			query := func() (time.Duration, error) {
				return untypedQuery(cfg.OutputDir(), "servergroups.json", objqry)
			}
			timedQuery(recorder, "servergroups", "", query)
		case "Nodes":
			// cfg.Nodes configures whether users want to gather the Nodes resource in the
			// cluster, but we also use that option to guide whether we get node data such
			// as configz and healthz endpoints.

			// NOTE: Node data collection is an aggregated time b/c propagating that detail back up
			// is odd and would pollute some of the output.
			start := time.Now()
			err := gatherNodeData(kubeClient, cfg)
			duration := time.Since(start)
			recorder.RecordQuery("Nodes", "", duration, err)
			fallthrough
		default:
			lister := func() (runtime.Object, error) { return queryNonNsResource(resourceKind, kubeClient) }
			query := func() (time.Duration, error) { return objListQuery(outdir+"/", resourceKind+".json", lister) }
			timedQuery(recorder, resourceKind, "", query)
		}
	}

	return nil
}
