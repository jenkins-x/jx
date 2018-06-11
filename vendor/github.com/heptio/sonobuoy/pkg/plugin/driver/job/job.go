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

package job

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/utils"
)

// Plugin is a plugin driver that dispatches a single pod to the given
// kubernetes cluster.
type Plugin struct {
	driver.Base
}

// Ensure Plugin implements plugin.Interface
var _ plugin.Interface = &Plugin{}

// NewPlugin creates a new DaemonSet plugin from the given Plugin Definition
// and sonobuoy master address.
func NewPlugin(dfn plugin.Definition, namespace, sonobuoyImage, imagePullPolicy string) *Plugin {
	return &Plugin{
		driver.Base{
			Definition:      dfn,
			SessionID:       utils.GetSessionID(),
			Namespace:       namespace,
			SonobuoyImage:   sonobuoyImage,
			ImagePullPolicy: imagePullPolicy,
			CleanedUp:       false, // be explicit
		},
	}
}

// ExpectedResults returns the list of results expected for this plugin. Since
// a Job only launches one pod, only one result type is expected.
func (p *Plugin) ExpectedResults(nodes []v1.Node) []plugin.ExpectedResult {
	return []plugin.ExpectedResult{
		plugin.ExpectedResult{ResultType: p.GetResultType()},
	}
}

func getMasterAddress(hostname string) string {
	return fmt.Sprintf("https://%s/api/v1/results/global", hostname)
}

//FillTemplate populates the internal Job YAML template with the values for this particular job.
func (p *Plugin) FillTemplate(hostname string, cert *tls.Certificate) ([]byte, error) {
	var b bytes.Buffer

	tmplData, err := p.GetTemplateData(getMasterAddress(hostname), cert)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get template data for %q", p.Definition.Name)
	}

	if err := jobTemplate.Execute(&b, tmplData); err != nil {
		return nil, errors.Wrapf(err, "couldn't fill template %q", p.Definition.Name)
	}

	return b.Bytes(), nil
}

// Run dispatches worker pods according to the Job's configuration.
func (p *Plugin) Run(kubeclient kubernetes.Interface, hostname string, cert *tls.Certificate) error {
	var job v1.Pod

	b, err := p.FillTemplate(hostname, cert)
	if err != nil {
		// Already wrapped sufficiently by FillTemplate
		return errors.Wrapf(err, "failed to fill Job template for plugin %v", p.GetName())
	}

	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), b, &job); err != nil {
		return errors.Wrapf(err, "could not decode executed template into a Job for plugin %v", p.GetName())
	}

	secret, err := p.MakeTLSSecret(cert)
	if err != nil {
		return errors.Wrapf(err, "couldn't make secret for Job plugin %v", p.GetName())
	}

	if _, err := kubeclient.CoreV1().Secrets(p.Namespace).Create(secret); err != nil {
		return errors.Wrapf(err, "couldn't create TLS secret for job plugin %v", p.GetName())
	}

	if _, err := kubeclient.CoreV1().Pods(p.Namespace).Create(&job); err != nil {
		return errors.Wrapf(err, "could not create Job resource for Job plugin %v", p.GetName())
	}

	return nil
}

// Monitor adheres to plugin.Interface by ensuring the pod created by the job
// doesn't have any urecoverable failures.
func (p *Plugin) Monitor(kubeclient kubernetes.Interface, _ []v1.Node, resultsCh chan<- *plugin.Result) {
	for {
		// Sleep between each poll, which should give the Job
		// enough time to create a Pod
		// TODO: maybe use a watcher instead of polling.
		time.Sleep(10 * time.Second)
		// If we've cleaned up after ourselves, stop monitoring
		if p.CleanedUp {
			break
		}

		// Make sure there's a pod
		pod, err := p.findPod(kubeclient)
		if err != nil {
			resultsCh <- utils.MakeErrorResult(p.GetResultType(), map[string]interface{}{"error": err.Error()}, "")
			break
		}

		// Make sure the pod isn't failing
		if isFailing, reason := utils.IsPodFailing(pod); isFailing {
			resultsCh <- utils.MakeErrorResult(p.GetResultType(), map[string]interface{}{
				"error": reason,
				"pod":   pod,
			}, "")
			break
		}
	}
}

// Cleanup cleans up the k8s Job and ConfigMap created by this plugin instance
func (p *Plugin) Cleanup(kubeclient kubernetes.Interface) {
	p.CleanedUp = true
	gracePeriod := int64(plugin.GracefulShutdownPeriod)
	deletionPolicy := metav1.DeletePropagationBackground

	listOptions := metav1.ListOptions{
		LabelSelector: "sonobuoy-run=" + p.GetSessionID(),
	}
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &deletionPolicy,
	}

	// Delete the Pod created by the job manually (just deleting the Job
	// doesn't kill the pod, it still lets it finish.)
	// TODO: for now we're not actually creating a Job at all, just a
	// single Pod, to get the restart semantics we want. But later if we
	// want to make this a real Job, we still need to delete pods manually
	// after deleting the job.
	err := kubeclient.CoreV1().Pods(p.Namespace).DeleteCollection(
		&deleteOptions,
		listOptions,
	)
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "error deleting pods for Job-%v", p.GetSessionID()))
	}
}

func (p *Plugin) listOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: "sonobuoy-run=" + p.GetSessionID(),
	}
}

// findPod finds the pod created by this plugin, using a kubernetes label
// search.  If no pod is found, or if multiple pods are found, returns an
// error.
func (p *Plugin) findPod(kubeclient kubernetes.Interface) (*v1.Pod, error) {
	pods, err := kubeclient.CoreV1().Pods(p.Namespace).List(p.listOptions())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(pods.Items) != 1 {
		return nil, errors.Errorf("no pods were created by plugin %v", p.Definition.Name)
	}

	return &pods.Items[0], nil
}
