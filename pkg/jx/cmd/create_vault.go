package cmd

import (
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
)

const supportedKubeProvider = "gke"

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

	options.addCommonFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreateVaultOptions) Run() error {
	teamSettings, err := o.TeamSettings()
	if err != nil {
		return errors.Wrap(err, "retrieving the team settings")
	}

	if teamSettings.KubeProvider != supportedKubeProvider {
		return errors.Wrapf(err, "this command only supports the '%s' kubernetes provider", supportedKubeProvider)
	}

	return nil
}
