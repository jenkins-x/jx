package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetOrgOrUserFromOptions_orgIsSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:  "MyOrg",
		User: "MyUser",
	}
	owner := GetOrgOrUserFromOptions(options)
	assert.Equal(t, options.Org, owner, "The Owner should be the Org name")
}

func TestGetOrgOrUserFromOptions_orgNotSetUserIsSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:  "",
		User: "MyUser",
	}
	owner := GetOrgOrUserFromOptions(options)
	assert.Equal(t, options.User, owner, "The Owner should be the Username")
}

func TestGetOrgOrUserFromOptions_orgNotSetUserNotSet(t *testing.T) {
	t.Parallel()
	options := &UpdateWebhooksOptions{
		Org:  "",
		User: "",
	}
	owner := GetOrgOrUserFromOptions(options)
	assert.Equal(t, "", owner, "The Owner should be empty")
}
