package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/helm/pkg/helm"

	"github.com/Azure/draft/pkg/draft/local"
	"github.com/Azure/draft/pkg/storage/kube/configmap"
)

const deleteDesc = `This command deletes an application from your Kubernetes environment.`

type deleteCmd struct {
	appName string
	out     io.Writer
}

func newDeleteCmd(out io.Writer) *cobra.Command {
	var runningEnvironment string

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
			return dc.run(runningEnvironment)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&runningEnvironment, environmentFlagName, environmentFlagShorthand, defaultDraftEnvironment(), environmentFlagUsage)

	return cmd
}

func (d *deleteCmd) run(runningEnvironment string) error {

	var name string

	if d.appName != "" {
		name = d.appName
	} else {
		deployedApp, err := local.DeployedApplication(draftToml, runningEnvironment)
		if err != nil {
			return errors.New("Unable to detect app name\nPlease pass in the name of the application")

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
	client, config, err := getKubeClient(kubeContext)
	if err != nil {
		return fmt.Errorf("Could not get a kube client: %s", err)
	}

	// delete Draft storage for app
	store := configmap.NewConfigMaps(client.CoreV1().ConfigMaps(tillerNamespace))
	if _, err := store.DeleteBuilds(context.Background(), app); err != nil {
		return err
	}

	helmClient, err := setupHelm(client, config, tillerNamespace)
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
