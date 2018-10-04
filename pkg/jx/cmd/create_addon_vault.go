package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

const (
	defaultVaultNamesapce   = "jx"
	defaultVaultReleaseName = "vault-operator"
)

var (
	CreateAddonVaultLong = templates.LongDesc(`
		Creates the Vault operator addon

		This addon will install an operator for HashiCorp Vault.""
`)

	CreateAddonVaultExample = templates.Examples(`
		# Create the vault-operator addon
		jx create addon vault-operator
	`)
)

// CreateAddonVaultptions the options for the create addon vault-operator
type CreateAddonVaultOptions struct {
	CreateAddonOptions
}

// NewCmdCreateAddonVault creates a command object for the "create addon vault-opeator" command
func NewCmdCreateAddonVault(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	commonOptions := CommonOptions{
		Factory: f,
		In:      in,
		Out:     out,
		Err:     errOut,
	}
	options := &CreateAddonVaultOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOptions,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "vault-operator",
		Short:   "Create an vault-operator addon for Hashicorp Vault",
		Long:    CreateAddonVaultLong,
		Example: CreateAddonVaultExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultVaultNamesapce, defaultVaultReleaseName)
	return cmd
}
