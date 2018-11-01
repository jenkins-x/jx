package vault_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/vault"
	"github.com/stretchr/testify/assert"
)

func TestEncodeVaultPathRule(t *testing.T) {
	tests := map[string]struct {
		rule *vault.PathRule
		err  bool
		want string
	}{
		"marshal policy": {
			rule: &vault.PathRule{
				Path: vault.PathPolicy{
					Prefix:       "secrets/*",
					Capabilities: []string{vault.CreateCapability, vault.ReadCapability},
				},
			},
			err:  false,
			want: "path \"secrets/*\" {  capabilities = [    \"create\",    \"read\",  ]}",
		},
		"marshal empty policy": {
			rule: &vault.PathRule{
				Path: vault.PathPolicy{
					Prefix:       "",
					Capabilities: []string{},
				},
			},
			err:  false,
			want: "path \"\" {  capabilities = []}",
		},
		"marshal nil policy with error": {
			rule: nil,
			err:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := tc.rule.String()
			if tc.err {
				assert.Error(t, err, "should encode policy with an error")
			} else {
				assert.NoError(t, err, "should encode policy without error")
			}
			assert.Equal(t, tc.want, output)
		})
	}
}
