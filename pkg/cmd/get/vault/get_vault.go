package vault

import (
	"reflect"

	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/io/secrets"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/kube/cluster"
	kubeVault "github.com/jenkins-x/jx/v2/pkg/kube/vault"
	"github.com/jenkins-x/jx/v2/pkg/vault"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type GetVaultOptions struct {
	*opts.CommonOptions

	Namespace           string
	DisableURLDiscovery bool
}

var (
	getVaultLong = templates.LongDesc(`
		Display Jenkins X system Vault as well as Vault instances created by 'jx create vault'.
	`)

	getVaultExample = templates.Examples(`
		# List all vaults 
		jx get vaults
	`)
)

// NewCmdGetVault creates a new command for 'jx get vaults'
func NewCmdGetVault(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetVaultOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "vault",
		Aliases: []string{"vaults"},
		Short:   "Display one or more Vaults",
		Long:    getVaultLong,
		Example: getVaultExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace from where to list the vaults")
	cmd.Flags().BoolVarP(&options.DisableURLDiscovery, "disableURLDiscovery", "", false, "Disables the automatic Vault URL discovery")
	return cmd
}

// Run implements the command
func (o *GetVaultOptions) Run() error {
	client, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return errors.Wrap(err, "creating kubernetes client")
	}

	if o.Namespace == "" {
		o.Namespace = ns
	}
	vaultOperatorClient, err := o.VaultOperatorClient()
	if err != nil {
		return errors.Wrap(err, "creating vault operator client")
	}

	var useIngressURL bool
	if o.DisableURLDiscovery {
		useIngressURL = true
	} else {
		useIngressURL = cluster.IsInCluster()
	}

	vaults, err := kubeVault.GetVaults(client, vaultOperatorClient, o.Namespace, useIngressURL)
	if err != nil {
		return errors.Wrap(err, "error retrieving Jenkins X managed Vault instances")
	}

	systemVault, err := o.systemVault(o.Namespace)
	if err != nil {
		return errors.Wrap(err, "error retrieving system Vault")
	}
	if systemVault != nil {
		vaults = o.appendUniq(vaults, systemVault)
	}

	table := o.CreateTable()
	table.AddRow("NAME", "URL", "AUTH-SERVICE-ACCOUNT")
	for _, v := range vaults {
		table.AddRow(v.Name, v.URL, v.ServiceAccountName)
	}
	table.Render()

	return nil
}

// systemVault gets the system vault
func (o *GetVaultOptions) systemVault(namespace string) (*vault.Vault, error) {
	kubeClient, devNamespace, err := o.CommonOptions.KubeClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create Kube client for Vault client creation")
	}

	if namespace != devNamespace {
		return nil, nil
	}

	installConfigMap, err := kubeClient.CoreV1().ConfigMaps(devNamespace).Get(kube.ConfigMapNameJXInstallConfig, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.Errorf("unable to determine Vault type since ConfigMap %s not found in namespace %s", kube.ConfigMapNameJXInstallConfig, devNamespace)
		}
		return nil, errors.Wrapf(err, "error retrieving ConfigMap %s in namespace %s", kube.ConfigMapNameJXInstallConfig, devNamespace)
	}

	installValues := installConfigMap.Data

	if installValues[secrets.SecretsLocationKey] != string(secrets.VaultLocationKind) {
		return nil, errors.Errorf("unable to create Vault client for secret location kind '%s'", installValues[secrets.SecretsLocationKey])
	}

	vault, err := vault.FromMap(installValues, devNamespace)
	return &vault, err
}

func (o *GetVaultOptions) appendUniq(vaults []*vault.Vault, newVault *vault.Vault) []*vault.Vault {
	for _, v := range vaults {
		if reflect.DeepEqual(v, newVault) {
			return vaults
		}
	}

	vaults = append(vaults, newVault)
	return vaults
}
