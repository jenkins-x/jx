package update

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/stretchr/testify/assert"
)

func TestGetOrgOrUserFromOptions_orgIsSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:           "MyOrg",
		CommonOptions: &opts.CommonOptions{Username: "MyUser"},
	}
	owner := GetOrgOrUserFromOptions(options)
	assert.Equal(t, options.Org, owner, "The Owner should be the Org name")
}

func TestGetOrgOrUserFromOptions_orgNotSetUserIsSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:           "",
		CommonOptions: &opts.CommonOptions{Username: "MyUser"},
	}
	owner := GetOrgOrUserFromOptions(options)
	assert.Equal(t, options.Username, owner, "The Owner should be the Username")
}

func TestGetOrgOrUserFromOptions_orgNotSetUserNotSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:           "",
		CommonOptions: &opts.CommonOptions{Username: ""},
	}
	owner := GetOrgOrUserFromOptions(options)
	assert.Equal(t, "", owner, "The Owner should be empty")
}
