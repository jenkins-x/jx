package cmd

import (
	"fmt"
	"io"

	"github.com/jenkins-x/jx/pkg/cloud/gke"
	gkevault "github.com/jenkins-x/jx/pkg/cloud/gke/vault"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/serviceaccount"
	kubevault "github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteVaultOptions keeps the options of delete vault command
type DeleteVaultOptions struct {
	CommonOptions

	Namespace            string
	RemoveCloudResources bool
	GKEProjectID         string
	GKEZone              string
}

var (
	deleteVaultLong = templates.LongDesc(`
		Deletes a Vault
	`)

	deleteVaultExample = templates.Examples(`
		# Deletes a Vault from namespace my-namespace
		jx delete vault --namespace my-namespace my-vault
	`)
)

// NewCmdDeleteVault builds a new delete vault command
func NewCmdDeleteVault(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteVaultOptions{
		CommonOptions: CommonOptions{
			Factory: f,
			In:      in,
			Out:     out,
			Err:     errOut,
		},
	}

	cmd := &cobra.Command{
		Use:     "vault",
		Short:   "Deletes a Vault",
		Long:    deleteVaultLong,
		Example: deleteVaultExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace from where to delete the vault")
	cmd.Flags().BoolVarP(&options.RemoveCloudResources, "remove-cloud-resources", "r", false, "Remove all cloud resource allocated for the Vault")
	cmd.Flags().StringVarP(&options.GKEProjectID, "gke-project-id", "", "", "Google Project ID to use for Vault backend")
	cmd.Flags().StringVarP(&options.GKEZone, "gke-zone", "", "", "The zone (e.g. us-central1-a) where Vault will store the encrypted data")
	return cmd
}

// Run implements the delete vault command
func (o *DeleteVaultOptions) Run() error {
	if len(o.Args) != 1 {
		return fmt.Errorf("Missing vault name")
	}
	vaultName := o.Args[0]

	client, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return errors.Wrap(err, "creating kubernetes client")
	}

	if o.Namespace == "" {
		o.Namespace = ns
	}

	clusterName, err := gke.ShortClusterName(o.Kube())
	if err != nil {
		return err
	}
	vaultOperatorClient, err := o.VaultOperatorClient()
	if err != nil {
		return errors.Wrap(err, "creating vault operator client")
	}

	v, err := kubevault.GetVault(vaultOperatorClient, vaultName, o.Namespace)
	if err != nil {
		return fmt.Errorf("vault '%s' not found in namespace '%s'", vaultName, o.Namespace)
	}

	err = kubevault.DeleteVault(vaultOperatorClient, vaultName, o.Namespace)
	if err != nil {
		return errors.Wrap(err, "deleting the vault resource")
	}

	err = kube.DeleteIngress(client, o.Namespace, vaultName)
	if err != nil {
		return errors.Wrapf(err, "deleting the vault ingress '%s'", vaultName)
	}

	authServiceAccountName := kubevault.GetAuthSaName(*v)
	err = serviceaccount.DeleteServiceAccount(client, o.Namespace, authServiceAccountName)
	if err != nil {
		return errors.Wrapf(err, "deleting the vault auth service account '%s'", authServiceAccountName)
	}

	gcpServiceAccountSecretName := gkevault.GcpServiceAccountSecretName(vaultName, clusterName)
	err = client.CoreV1().Secrets(o.Namespace).Delete(gcpServiceAccountSecretName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "deleting secret '%s' where GCP service account is stored", gcpServiceAccountSecretName)
	}

	err = kube.DeleteClusterRoleBinding(client, vaultName)
	if err != nil {
		return errors.Wrapf(err, "deleting the cluster role binding '%s' for vault", vaultName)
	}

	log.Infof("Vault %s deleted\n", util.ColorInfo(vaultName))

	if o.RemoveCloudResources {
		teamSettings, err := o.TeamSettings()
		if err != nil {
			return errors.Wrap(err, "retrieving the team settings")
		}

		if teamSettings.KubeProvider == gkeKubeProvider {
			log.Infof("Removing GCP resources allocated for Vault...\n")
			err := o.removeGCPResources(vaultName)
			if err != nil {
				return errors.Wrap(err, "removing GCP resource")
			}
			log.Infof("Cloud resources allocated for vault %s deleted\n", util.ColorInfo(vaultName))
		}

	}

	return nil
}

func (o *DeleteVaultOptions) removeGCPResources(vaultName string) error {
	err := gke.Login("", true)
	if err != nil {
		return errors.Wrap(err, "login into GCP")
	}

	if o.GKEProjectID == "" {
		projectId, err := o.getGoogleProjectId()
		if err != nil {
			return err
		}
		o.GKEProjectID = projectId
	}
	err = o.runCommandVerbose("gcloud", "config", "set", "project", o.GKEProjectID)
	if err != nil {
		return err
	}

	if o.GKEZone == "" {
		zone, err := o.getGoogleZone(o.GKEProjectID)
		if err != nil {
			return err
		}
		o.GKEZone = zone
	}

	clusterName, err := gke.ShortClusterName(o.Kube())
	if err != nil {
		return err
	}

	sa := gkevault.ServiceAccountName(vaultName, clusterName)
	err = gke.DeleteServiceAccount(sa, o.GKEProjectID, gkevault.ServiceAccountRoles)
	if err != nil {
		return errors.Wrapf(err, "deleting the GCP service account '%s'", sa)
	}
	log.Infof("GCP service account %s deleted\n", util.ColorInfo(sa))

	bucket := gkevault.BucketName(vaultName, clusterName)
	err = gke.DeleteAllObjectsInBucket(bucket)
	if err != nil {
		return errors.Wrapf(err, "deleting all objects in GCS bucket '%s'", bucket)
	}

	log.Infof("GCS bucket %s deleted\n", util.ColorInfo(bucket))

	return nil
}
