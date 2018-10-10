package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
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

	if o.Namespace == "" {
		_, ns, err := o.KubeClient()
		if err != nil {
			o.Namespace = ns
		}
	}

	err = gke.Login("", false)
	if err != nil {
		return errors.Wrap(err, "login into GCP")
	}

	secretName, err := o.createVaultGCPServiceAccount()
	if err != nil {
		return err
	}
	fmt.Println(secretName)

	return nil
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
