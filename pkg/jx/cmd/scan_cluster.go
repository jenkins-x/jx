package cmd

import (
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kubeHunterImage         = "cosmincojocar/kube-hunter:latest"
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

	container := o.hunterContainer()
	job := o.createScanJob(kubeHunterJobName, ns, container)

	_, err = kubeClient.BatchV1().Jobs(ns).Create(job)
	if err != nil {
		return err
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
