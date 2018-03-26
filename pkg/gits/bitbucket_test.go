package gits

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/stretchr/testify/assert"
)

func TestListOrganisations(t *testing.T) {

	as := auth.AuthServer{
		URL:         "https://auth.example.com",
		Name:        "Test Auth Server",
		Kind:        "Oauth2",
		CurrentUser: "wbrefvem",
	}
	ua := auth.UserAuth{
		Username: "wbrefvem",
		ApiToken: "0123456789abdef",
	}

	bp, err := NewBitbucketProvider(&as, &ua)

	assert.Nil(t, err)
	assert.NotNil(t, bp)
}
