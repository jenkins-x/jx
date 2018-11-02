package cmd

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/jx/cmd/vault"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"runtime"
)

type GetVaultConfigOptions struct {
	GetOptions

	Namespace string
}

var (
	getVaultConfigLong = templates.LongDesc(`
		Echoes the configuration required for connecting to a vault using the official vault CLI client	
	`)

	getVaultConfigExample = templates.Examples(`
		# Gets vault config
		jx get vault-config
	`)
)

// NewCmdGetVaultConfig creates a new command for 'jx get secrets'
func NewCmdGetVaultConfig(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &GetVaultConfigOptions{
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
		Use:     "vault-config",
		Short:   "Gets the configuration for using the official vault CLI client",
		Long:    getVaultConfigLong,
		Example: getVaultConfigExample,
		Run: func(c *cobra.Command, args []string) {
			options.Cmd = c
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addGetFlags(cmd)

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace from where to get the vault config")
	cmd.Flags().StringVarP(&options.Namespace, "name", "m", "", "Name of the vault to get the config for")
	return cmd
}

// Run implements the command
func (o *GetVaultConfigOptions) Run() error {
	clientFactory := vault.VaultClientFactory{
		Options: o,
	}
	client, err := clientFactory.NewVaultClient(o.Namespace)
	if err != nil {
		return err
	}

	// Echo the client config out to the command line to be piped into bash
	if runtime.GOOS == "windows" {
		fmt.Fprintf(o.Out, "set VAULT_ADDR=\"%s\"&&set VAULT_TOKEN=\"%s\"\n", client.Address(), client.Token())
	} else {
		fmt.Fprintf(o.Out, "export VAULT_ADDR=%s && export VAULT_TOKEN=%s\n", client.Address(), client.Token())
	}

	return nil
}
