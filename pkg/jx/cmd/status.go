package cmd

import (
	"io"

	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type StatusOptions struct {
	CommonOptions
	node string
}

var (
	StatusLong = templates.LongDesc(`
		Gets the current status of the Kubernetes cluster

`)

	StatusExample = templates.Examples(`
		# displays the current status of the kubernetes cluster
		jx status
`)
)

func NewCmdStatus(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &StatusOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			Out:     out,
			Err:     errOut,
		},
	}
	cmd := &cobra.Command{
		Use:     "status [node]",
		Short:   "status of the Kubernetes cluster or named node",
		Long:    StatusLong,
		Example: StatusExample,
		Aliases: []string{"status"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.node, "node", "n", "", "the named node to get ")
	return cmd
}

func (o *StatusOptions) Run() error {

	client, namespace, err := o.KubeClient()
	if err != nil {

		log.Warn("Unable to connect to Kubernetes cluster -  is one running ?")
		log.Warn("you could try: jx create cluster - e.g: " + createClusterExample + "\n\n")
		log.Warn(createClusterLong)

		return err
	}

	/*
	 * get status for all pods in all namespaces
	 */
	clusterStatus, err := kube.GetClusterStatus(client, "")
	if err != nil {
		log.Error("Failed to get cluster status " + err.Error() + " \n")
		return err
	}

	deployList, err := client.ExtensionsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Error("Failed to get deployed  status " + err.Error() + " \n")
		return err
	}

	if deployList == nil || len(deployList.Items) == 0 {
		log.Warnf("Unable to find JX components in %s", clusterStatus.Info())
		log.Info("you could try: " + instalExample + "\n\n")
		log.Info(instalLong)
		return fmt.Errorf("no deployments found in namespace %s", namespace)
	}

	for _, d := range deployList.Items {
		err = kube.WaitForDeploymentToBeReady(client, d.Name, namespace, 5*time.Second)
		if err != nil {
			log.Warnf("%s: jx deployment %s not ready in namespace %s", clusterStatus.Info(), d.Name, namespace)
			return err
		}
	}
	resourceStr := clusterStatus.CheckResource()

	jenkinsURL, err := o.findServiceInNamespace("jenkins", namespace)
	if err != nil {
		if resourceStr != "" {
			log.Warnf("%s Jenkins not found and %s\n", clusterStatus.Info(), resourceStr)
		} else {
			log.Warnf("%s Jenkins not found\n", clusterStatus.Info())
		}
		return err
	}
	if resourceStr != "" {
		log.Warnf("Jenkins X installed for %s. Jenkins is running at %s. Need more %s\n", clusterStatus.Info(), jenkinsURL, resourceStr)
	} else {
		log.Successf("Jenkins X checks passed for %s. Jenkins is running at %s\n", clusterStatus.Info(), jenkinsURL)
	}

	return nil
}
