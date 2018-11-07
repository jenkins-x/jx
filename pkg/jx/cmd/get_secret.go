package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
)

type GetSecretOptions struct {
	GetOptions

	Namespace string
	Name      string
}

var (
	getSecretLong = templates.LongDesc(`
		Display one or more Vault Secrets	
	`)

	getSecretExample = templates.Examples(`
		# List all secrets
		jx get secrets
	`)
)

// NewCmdGetSecret creates a new command for 'jx get secrets'
func NewCmdGetSecret(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetSecretOptions{
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
		Use:     "secrets",
		Short:   "Display one or more Secrets",
		Long:    getSecretLong,
		Example: getSecretExample,
		Run: func(c *cobra.Command, args []string) {
			options.Cmd = c
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addGetFlags(cmd)

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace from where to list the secrets")
	cmd.Flags().StringVarP(&options.Namespace, "name", "m", "", "The name of the Vault to use")
	return cmd
}

// Run implements the command
func (o *GetSecretOptions) Run() error {
	clientFactory := vault.VaultClientFactory{
		Options: o,
	}
	client, err := clientFactory.NewVaultClient(o.Name, o.Namespace)
	if err != nil {
		return err
	}
	secrets, err := vault.GetSecrets(client)
	if err != nil {
		return err
	}

	table := o.CreateTable()
	table.AddRow("KEY")
	for _, secret := range secrets {
		table.AddRow(secret)
	}
	table.Render()

	return nil
}
