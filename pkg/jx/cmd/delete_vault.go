package cmd

import (
	"io"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type DeleteVaultOptions struct {
	CommonOptions

	Namespace string
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
	return cmd
}

func (o *DeleteVaultOptions) Run() error {
	return nil
}
