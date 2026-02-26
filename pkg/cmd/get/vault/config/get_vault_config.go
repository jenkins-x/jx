package config

import (
	"fmt"
	"runtime"

	"github.com/jenkins-x/jx/v2/pkg/vault"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/spf13/cobra"
)

type GetVaultConfigOptions struct {
	*opts.CommonOptions

	Namespace string
	Name      string
	terminal  string
}

var (
	getVaultConfigLong = templates.LongDesc(`
Used to echo the Vault connection configuration for the Jenkins X system Vault.
To have the settings apply to the current terminal session the output must be evaluated, for example:

$ eval $(jx get vault-config)

Together with the name and namespace option, this command can be used to echo the connection configuration
for any vault installed via 'jx add vault'.
	`)

	getVaultConfigExample = templates.Examples(`
		# Gets vault config
		jx get vault-config
	`)
)

// NewCmdGetVaultConfig creates a new command for 'jx get secrets'
func NewCmdGetVaultConfig(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &GetVaultConfigOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "vault-config",
		Short:   "Gets the configuration for using the Vault CLI",
		Long:    getVaultConfigLong,
		Example: getVaultConfigExample,
		Run: func(c *cobra.Command, args []string) {
			options.Cmd = c
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace from where to get the Vault config")
	cmd.Flags().StringVarP(&options.Name, "name", "m", "", "Name of the Vault to get the config for")
	cmd.Flags().StringVarP(&options.terminal, "terminal", "t", "", "terminal type output override. Values: ['sh', 'cmd'].")
	return cmd
}

// Run implements the command
func (o *GetVaultConfigOptions) Run() error {
	var vaultClient vault.Client
	var err error

	if o.Name != "" || o.Namespace != "" {
		vaultClient, err = o.vaultClient(o.Name, o.Namespace)
		if err != nil {
			return err
		}
	} else {
		vaultClient, err = o.systemVaultClient()
		if err != nil {
			return err
		}
	}

	url, token, err := vaultClient.Config()
	// Echo the client config out to the command line to be piped into bash
	if o.terminal == "" {
		if runtime.GOOS == "windows" {
			o.terminal = "cmd"
		} else {
			o.terminal = "sh"
		}
	}
	if o.terminal == "cmd" {
		_, _ = fmt.Fprintf(o.Out, "set VAULT_ADDR=%s\nset VAULT_TOKEN=%s\n", url.String(), token)
	} else {
		_, _ = fmt.Fprintf(o.Out, "export VAULT_ADDR=%s\nexport VAULT_TOKEN=%s\n", url.String(), token)
	}

	return err
}

func (o *GetVaultConfigOptions) systemVaultClient() (vault.Client, error) {
	_, devNamespace, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create Kube client")
	}

	return o.SystemVaultClient(devNamespace)
}

func (o *GetVaultConfigOptions) vaultClient(name string, namespace string) (vault.Client, error) {
	factory := o.GetFactory()
	client, err := factory.CreateInternalVaultClient(name, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create Vault client for Jenkins X managed Vault instance")
	}

	return client, nil
}
