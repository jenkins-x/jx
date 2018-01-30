package kube

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"

	metricsclient "k8s.io/metrics/pkg/client/clientset_generated/clientset"

)

/*
func GetPodMetrics(kubeClient *kubernetes.Clientset, ns string) ([]byte, error) {
	path := util.UrlJoin("/proxy/apis/metrics/v1alpha1/namespaces/", ns, "/pods")
	params := map[string]string{}
	// TODO we should use some object to cache if we have a heapster and what scheme/port its on?
	resp := kubeClient.CoreV1().Services("kube-system").ProxyGet("http", "heapster", "80", path, params)
	return resp.DoRaw()
}
*/

func GetPodMetrics(client *metricsclient.Clientset, ns string) (*metricsv1beta1.PodMetricsList, error) {
	return client.MetricsV1beta1().PodMetricses(ns).List(metav1.ListOptions{})
}
