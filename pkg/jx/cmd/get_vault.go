package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/kube/vault"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
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
func NewCmdGetVault(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetVaultOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
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
			CheckErr(err)
		},
	}

	options.addGetFlags(cmd)

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace from where to list the vaults")
	return cmd
}

// Run implements the command
func (o *GetVaultOptions) Run() error {
	client, ns, err := o.KubeClient()
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
		return err
	}

	table := o.createTable()
	table.AddRow("NAME", "URL", "AUTH-SERVICE-ACCOUNT")
	for _, vault := range vaults {
		table.AddRow(vault.Name, vault.URL, vault.AuthServiceAccountName)
	}
	table.Render()

	return nil
}
