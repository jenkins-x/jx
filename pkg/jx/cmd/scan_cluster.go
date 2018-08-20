package cmd

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kubeHunterImage         = "cosmincojocar/kube-hunter:v20"
	kubeHunterContainerName = "jx-kube-hunter"
	kubeHunterNamespace     = "jx-kube-hunter"
	kubeHunterJobName       = "jx-kube-hunter-job"
)

// ScanClusterOptions the options for 'scan cluster' command
type ScanClusterOptions struct {
	ScanOptions
}

// NewCmdScanCluster creates a command object for "scan cluster" command
func NewCmdScanCluster(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &ScanClusterOptions{
		ScanOptions: ScanOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Performs a cluster security scan",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	return cmd
}

// Run executes the "scan cluster" command
func (o *ScanClusterOptions) Run() error {
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "creating kube client")
	}

	// Create a dedicated namespace for kube-hunter scan
	ns := kubeHunterNamespace
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	_, err = kubeClient.CoreV1().Namespaces().Create(namespace)
	if err != nil {
		return errors.Wrapf(err, "creating namespace '%s'", ns)
	}

	// Start the kube-hunter scanning
	container := o.hunterContainer()
	job := o.createScanJob(kubeHunterJobName, ns, container)
	job, err = kubeClient.BatchV1().Jobs(ns).Create(job)
	if err != nil {
		return err
	}

	// Wait for scanning to complete successfully
	log.Info("Waiting for kube hunter job to complete the scanning...\n")
	err = kube.WaitForJobToSucceeded(kubeClient, ns, kubeHunterJobName, 3*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waiting for kube hunter job to complete the scanning")
	}

	result, err := o.retriveScanResult(ns)
	if err != nil {
		return errors.Wrap(err, "retrieving scan results")
	}
	log.Info(result)

	// Clean up the kube-hunter namespace
	err = kubeClient.CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "cleaning up the scanning namespace '%s'", ns)
	}

	return nil
}

func (o *ScanClusterOptions) hunterContainer() *v1.Container {
	return &v1.Container{
		Name:            kubeHunterContainerName,
		Image:           kubeHunterImage,
		ImagePullPolicy: v1.PullIfNotPresent,
		Command:         []string{"python", "kube-hunter.py"},
		Args:            []string{"--pod", "--report=yaml", "--log=none"},
	}
}

func (o *ScanClusterOptions) createScanJob(name string, namespace string, container *v1.Container) *batchv1.Job {
	podTmpl := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Containers:    []v1.Container{*container},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: podTmpl,
		},
	}
}

func (o *ScanClusterOptions) retriveScanResult(namespace string) (string, error) {
	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return "", errors.Wrap(err, "creating kube client")
	}

	labels := map[string]string{"job-name": kubeHunterJobName}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})
	podList, err := kubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return "", errors.Wrap(err, "listing the scan job PODs")
	}
	foundPods := len(podList.Items)
	if foundPods != 1 {
		return "", fmt.Errorf("one POD expected for security scan job '%s'. Found: %d PODs.", kubeHunterJobName, foundPods)
	}

	podName := podList.Items[0].Name
	logOpts := &v1.PodLogOptions{
		Container: kubeHunterContainerName,
		Follow:    false,
	}
	req := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, logOpts)
	readCloser, err := req.Stream()
	if err != nil {
		return "", errors.Wrap(err, "creating the logs stream reader")
	}
	defer readCloser.Close()

	var result []byte
	reader := bufio.NewReader(readCloser)
	for {
		line, _, err := reader.ReadLine()
		if err != nil && err != io.EOF {
			return "", errors.Wrapf(err, "reading logs from POD  '%s'", podName)
		}
		if err == io.EOF {
			break
		}
		line = append(line, '\n')
		result = append(result, line...)
	}

	return string(result), nil
}
