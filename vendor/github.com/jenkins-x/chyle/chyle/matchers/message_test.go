package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemovePGPKey(t *testing.T) {
	type g struct {
		originalValue string
		expected      string
	}

	tests := []g{
		{
			" kDBcBAABCAAGBQJYRXDpAAoJEP6+b85LBShD46IP+gNG/HO+BMDm76SZVIIkOMnS\n qw/DhStoxYgdjbwKxiMVISQ5fHJqX4+PbRv2pcMtWy74cK79qK1OgxLCWWLf4zft\n FiRfp/Wq92ChglsN95GI0IrbehloqdP1wzSMo99WtNGb8uacsnO1P9pDF6PzITUn\n nhpCmIek6AUP5iUZ5E1lF2QuTbc9zM3q5Lq0G2RUZ9AGQNM5HUEql6/zvsqlx5Xl\n gdPf9daDFB6W/rpFEAU5VskcUTYKgSKqvpNDE13Hz56F4J0Z4mRb7fUk6bMqW8bH\n 68cWhbRV55qgdNDoIHMaQavexkBeR5sZ/Czs8/ajNS6Z3hv7qy57YMw3Z5Hqkn57\n 3JKmM56YpkhBM6eQTEoRjWRMOeI8QlxgaXrLPB8WZzf9J7E7R85MWF7iuWxeBHAq\n Fo2aK35Sbxa2hsZUv+cCH7tJHtDSTnSgORC+vXeBL7PzLKYQ1fwJA0buJBdU+CNX\n 8SyDoOR44u58HksxUZecqXKgOTyyJer5hkGY8IlxIBaqLDkV/TyDKQCHCqNTAi7a\n DTYG+qvTVBnFuRv3vaYOMALKsiEFQPUtEK+lLc/TGlfyp4hSY3VC6Gggx4WUUPG+\n Mb+FdfpuEVPp/lBMcIIveolM29Pf66Cs/bYoJoFC/lbkBKBAEdE4PlUC9l0S7gLF\n xVbg93wF3uLMJtF63j0f\n =IaBk\n -----END PGP SIGNATURE-----\n\ntest :whatever\n",
			"test :whatever\n",
		},
		{
			"test\n\ntest : whatever\n",
			"test\n\ntest : whatever\n",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, removePGPKey(test.originalValue))
	}
}
