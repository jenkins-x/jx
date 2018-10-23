package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	gkeKubeProvider            = "gke"
	gkeServiceAccountSecretKey = "service-account.json"
)

var (
	vaultServiceAccountRoles = []string{"roles/storage.objectAdmin",
		"roles/cloudkms.admin",
		"roles/cloudkms.cryptoKeyEncrypterDecrypter",
	}
)

var (
	createVaultLong = templates.LongDesc(`
		Creates a Vault using the vault-operator
`)

	createVaultExample = templates.Examples(`
		# Create a new vault 
		jx create vault
"
	`)
)

// CreateVaultOptions the options for the create vault command
type CreateVaultOptions struct {
	CreateOptions

	GKEProjectID string
	GKEZone      string
	Namespace    string
}

// NewCmdCreateVault  creates a command object for the "create" command
func NewCmdCreateVault(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateVaultOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "vault",
		Short:   "Create a new Vault using the vault-opeator",
		Long:    createVaultLong,
		Example: createVaultExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.GKEProjectID, "gke-project-id", "", "", "Google Project ID to use for Vault backend")
	cmd.Flags().StringVarP(&options.GKEZone, "gke-zone", "", "", "The zone (e.g. us-central1-a) where Vault will store the encrypted data")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace where the Vault is created")

	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreateVaultOptions) Run() error {
	teamSettings, err := o.TeamSettings()
	if err != nil {
		return errors.Wrap(err, "retrieving the team settings")
	}

	if teamSettings.KubeProvider != gkeKubeProvider {
		return errors.Wrapf(err, "this command only supports the '%s' kubernetes provider", gkeKubeProvider)
	}

	return o.createVaultGKE()
}

func (o *CreateVaultOptions) createVaultGKE() error {
	_, team, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "creating kubernetes client")
	}

	if o.Namespace == "" {
		o.Namespace = team
	}

	err = gke.Login("", false)
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

	err = o.CreateOptions.CommonOptions.runCommandVerbose(
		"gcloud", "config", "set", "project", o.GKEProjectID)
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

	log.Infof("Creating GCP service account for Vault backend\n")
	gcpServiceAccountSecretName, err := o.createVaultGCPServiceAccount()
	if err != nil {
		return errors.Wrap(err, "creating GCP service account")
	}
	log.Infof("%s service account created\n", util.ColorInfo(gcpServiceAccountSecretName))

	log.Infof("Setting up GCP KMS configuration\n")
	kmsConfig, err := o.createKmsConfig(team)
	if err != nil {
		return errors.Wrap(err, "creating KMS configuration")
	}
	log.Infof("KMS Key %s created in keying %s\n", util.ColorInfo(kmsConfig.key), util.ColorInfo(kmsConfig.keyring))

	vaultBucket, err := o.createVaultBucket(team)
	if err != nil {
		return errors.Wrap(err, "creating Vault GCS data bucket")
	}
	log.Infof("GCS bucket %s was created for Vault backend\n", util.ColorInfo(vaultBucket))
	vaultAuthServiceAccount, err := o.createVaultAuthServiceAccount()
	if err != nil {
		return errors.Wrap(err, "creating Vault authentication service account")
	}
	log.Infof("Created service account %s for Vault authentication\n", util.ColorInfo(vaultAuthServiceAccount))

	log.Infof("Creating Vault...\n")
	vaultOperatorClient, err := o.VaultOperatorClient()
	if err != nil {
		return errors.Wrap(err, "creating vault opeator client")
	}

	vaultName := fmt.Sprintf("%s-vault", team)
	gcpConfig := &kube.GCPConfig{
		ProjectId:   o.GKEProjectID,
		KmsKeyring:  kmsConfig.keyring,
		KmsKey:      kmsConfig.key,
		KmsLocation: kmsConfig.location,
		GcsBucket:   vaultBucket,
	}
	err = kube.CreateVault(vaultOperatorClient, vaultName, o.Namespace, gcpServiceAccountSecretName,
		gcpConfig, vaultAuthServiceAccount, o.Namespace)
	if err != nil {
		return errors.Wrap(err, "creating vault")
	}

	log.Infof("Vault %s created\n", util.ColorInfo(vaultName))

	log.Infof("Exposing %s Vault...\n", util.ColorInfo(vaultName))
	return o.exposeVault(vaultName)
}

func (o *CreateVaultOptions) createVaultGCPServiceAccount() (string, error) {
	serviceAccountDir, err := ioutil.TempDir("/tmp", gkeKubeProvider)
	if err != nil {
		return "", errors.Wrap(err, "creating a temporary folder where the service account will be stored")
	}
	defer os.RemoveAll(serviceAccountDir)

	serviceAccountName, err := o.serviceAccountName()
	if err != nil {
		return "", err
	}
	serviceAccountPath, err := gke.GetOrCreateServiceAccount(serviceAccountName, o.GKEProjectID, serviceAccountDir, vaultServiceAccountRoles)
	if err != nil {
		return "", errors.Wrap(err, "creating the service account")
	}

	secretName, err := o.storeGCPServiceAccountIntoSecret(serviceAccountPath)
	if err != nil {
		return "", errors.Wrap(err, "storing the service account into a secret")
	}
	return secretName, nil
}

func (o *CreateVaultOptions) serviceAccountName() (string, error) {
	_, currentTeam, err := o.KubeClient()
	if err != nil {
		return "", errors.Wrap(err, "retrieving the current team name")
	}
	return fmt.Sprintf("%s-vault", currentTeam), nil
}

func (o *CreateVaultOptions) storeGCPServiceAccountIntoSecret(serviceAccountPath string) (string, error) {
	client, currentTeam, err := o.KubeClient()
	if err != nil {
		return "", errors.Wrap(err, "creating kubernetes client")
	}
	serviceAccount, err := ioutil.ReadFile(serviceAccountPath)
	if err != nil {
		return "", errors.Wrapf(err, "reading the service account from file '%s'", serviceAccountPath)
	}

	secretName := fmt.Sprintf("%s-vault-gcp-sa", currentTeam)
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Data: map[string][]byte{
			gkeServiceAccountSecretKey: serviceAccount,
		},
	}

	secrets := client.CoreV1().Secrets(o.Namespace)
	_, err = secrets.Get(secretName, metav1.GetOptions{})
	if err != nil {
		_, err = secrets.Create(secret)
	} else {
		_, err = secrets.Update(secret)
	}
	return secretName, nil
}

type kmsConfig struct {
	keyring  string
	key      string
	location string
	project  string
}

func (o *CreateVaultOptions) createKmsConfig(team string) (*kmsConfig, error) {
	config := &kmsConfig{
		keyring:  fmt.Sprintf("%s-vault-keyring", team),
		key:      fmt.Sprintf("%s-vault-key", team),
		location: gke.KmsLocation,
		project:  o.GKEProjectID,
	}

	err := gke.CreateKmsKeyring(config.keyring, config.project)
	if err != nil {
		return nil, errors.Wrapf(err, "creating kms keyring '%s'", config.keyring)
	}

	err = gke.CreateKmsKey(config.key, config.keyring, config.project)
	if err != nil {
		return nil, errors.Wrapf(err, "crating the kms key '%s'", config.key)
	}
	return config, nil
}

func (o *CreateVaultOptions) createVaultBucket(team string) (string, error) {
	bucketName := fmt.Sprintf("%s-vault-bucket", team)
	exists, err := gke.BucketExists(o.GKEProjectID, bucketName)
	if err != nil {
		return "", errors.Wrap(err, "checking if Vault GCS bucket exists")
	}
	if exists {
		return bucketName, nil
	}

	if o.GKEZone == "" {
		return "", errors.New("GKE zone must be provided in 'gke-zone' option")
	}
	region := gke.GetRegionFromZone(o.GKEZone)
	err = gke.CreateBucket(o.GKEProjectID, bucketName, region)
	if err != nil {
		return "", errors.Wrap(err, "creating Vault GCS bucket")
	}
	return bucketName, nil
}

func (o *CreateVaultOptions) createVaultAuthServiceAccount() (string, error) {
	client, team, err := o.KubeClient()
	if err != nil {
		return "", errors.Wrap(err, "creating kubernetes client")
	}

	serviceAccountName := fmt.Sprintf("%s-vault-auth-sa", team)
	_, err = kube.CreateServiceAccount(client, o.Namespace, serviceAccountName)
	if err != nil {
		return "", errors.Wrap(err, "creating vault auth service account")
	}
	return serviceAccountName, nil
}

func (o *CreateVaultOptions) exposeVault(vaultService string) error {
	err := kube.WaitForService(o.KubeClientCached, vaultService, o.Namespace, 1*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waiting for vault service")
	}
	svc, err := o.KubeClientCached.CoreV1().Services(o.Namespace).Get(vaultService, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "getting the vault service: %s", vaultService)
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	if svc.Annotations[kube.AnnotationExpose] == "" {
		svc.Annotations[kube.AnnotationExpose] = "true"
		svc, err = o.KubeClientCached.CoreV1().Services(o.Namespace).Update(svc)
		if err != nil {
			return errors.Wrap(err, "updating the service annotations")
		}
	}
	devNamespace, _, err := kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
	if err != nil {
		return errors.Wrap(err, "retrieving the dev namespace")
	}
	return o.exposeService(vaultService, devNamespace, o.Namespace)
}
