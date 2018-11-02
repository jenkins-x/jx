package vault_test

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/jenkins-x/jx/pkg/jx/cmd/vault"
	"testing"
)

func Test_hack(t *testing.T) {
	options := cmd.NewCommonOptions("", cmd.NewFactory())

	f := vault.VaultClientFactory{
		Options: &options,
	}
	client := f.NewVaultClient()

	t.Log(client.Address())
	t.Log(client.Token())
	t.Log(client.Logical().List("secret"))
}
