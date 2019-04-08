package cmd

import (
	"fmt"
	"time"

	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StatusOptions struct {
	*opts.CommonOptions
	node string
}

var (
	StatusLong = templates.LongDesc(`
		Gets the current status of the Kubernetes cluster

`)

	StatusExample = templates.Examples(`
		# displays the current status of the Kubernetes cluster
		jx status
`)
)

func NewCmdStatus(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StatusOptions{
		CommonOptions: commonOpts,
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
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.node, "node", "n", "", "the named node to get ")
	return cmd
}

func (o *StatusOptions) Run() error {

	client, namespace, err := o.KubeClientAndNamespace()
	if err != nil {

		log.Warn("Unable to connect to Kubernetes cluster -  is one running ?")
		log.Warn("you could try: jx create cluster - e.g: " + createClusterExample + "\n\n")
		log.Warn(createClusterLong)

		return err
	}

	/*
	 * get status for all pods in all namespaces
	 */
	clusterStatus, err := kube.GetClusterStatus(client, "", o.Verbose)
	if err != nil {
		log.Error("Failed to get cluster status " + err.Error() + " \n")
		return err
	}

	deployList, err := client.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
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

	if resourceStr != "" {
		log.Warnf("Jenkins X installed for %s.\n%s\n", clusterStatus.Info(), util.ColorWarning(resourceStr))
	} else {
		log.Successf("Jenkins X checks passed for %s.\n", clusterStatus.Info())
	}

	return nil
}
