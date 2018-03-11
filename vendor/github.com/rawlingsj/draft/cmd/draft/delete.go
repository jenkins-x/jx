package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"k8s.io/helm/pkg/helm"

	"github.com/Azure/draft/pkg/draft/local"
)

const deleteDesc = `This command deletes an application from your Kubernetes environment.`

type deleteCmd struct {
	appName string
	out     io.Writer
}

func newDeleteCmd(out io.Writer) *cobra.Command {
	dc := &deleteCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "delete [app]",
		Short: "delete an application",
		Long:  deleteDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				dc.appName = args[0]
			}
			return dc.run()
		},
	}

	return cmd
}

func (d *deleteCmd) run() error {

	var name string

	if d.appName != "" {
		name = d.appName
	} else {
		deployedApp, err := local.DeployedApplication(draftToml, defaultDraftEnvironment())
		if err != nil {
			return errors.New("Unable to detect app name\nPlesae pass in the name of the application")

		}

		name = deployedApp.Name
	}

	//TODO: replace with serverside call
	if err := Delete(name); err != nil {
		return err
	}

	msg := "app '" + name + "' deleted"
	fmt.Fprintln(d.out, msg)
	return nil
}

// Delete uses the helm client to delete an app with the given name
//
// Returns an error if the command failed.
func Delete(app string) error {
	// set up helm client
	client, clientConfig, err := getKubeClient(kubeContext)
	if err != nil {
		return fmt.Errorf("Could not get a kube client: %s", err)
	}
	restClientConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("Could not retrieve client config from the kube client: %s", err)
	}

	helmClient, err := setupHelm(client, restClientConfig, draftNamespace)
	if err != nil {
		return err
	}

	// delete helm release
	_, err = helmClient.DeleteRelease(app, helm.DeletePurge(true))
	if err != nil {
		return errors.New(grpc.ErrorDesc(err))
	}

	return nil
}
