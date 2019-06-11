package get

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/kube/vault"
	"github.com/jenkins-x/jx/pkg/log"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type GetVaultOptions struct {
	GetOptions

	Namespace string
}

var (
	getVaultLong = templates.LongDesc(`
		Display one or more vaults	
	`)

	getVaultExample = templates.Examples(`
		# List all vaults 
		jx get vaults
	`)
)

// NewCmdGetVault creates a new command for 'jx get vaults'
func NewCmdGetVault(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetVaultOptions{
		GetOptions: GetOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "vaults",
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

	options.AddGetFlags(cmd)

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace from where to list the vaults")
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

	vaults, err := vault.GetVaults(client, vaultOperatorClient, o.Namespace)
	if err != nil {
		log.Logger().Infof("No vault found.")
		return nil
	}

	table := o.CreateTable()
	table.AddRow("NAME", "URL", "AUTH-SERVICE-ACCOUNT")
	for _, vault := range vaults {
		table.AddRow(vault.Name, vault.URL, vault.AuthServiceAccountName)
	}
	table.Render()

	return nil
}
