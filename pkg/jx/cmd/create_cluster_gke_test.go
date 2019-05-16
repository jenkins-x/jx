package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_sanitizeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		want     string
	}{
		{"Replaces . in username for -", "test.person", "test-person"},
		{"Replaces _ in username for -", "test_person", "test-person"},
		{"Replaces uppercase in username for lowercase", "Test", "test"},
		{"Doesn't do anything for empty user names", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, sanitizeLabel(tt.username), tt.want)
		})
	}
}

func Test_validateClusterName(t *testing.T) {
	var bigLongName = string("this-name-is-too-long-to-be-used-by-2chars")
	var capitalName = string("NameWithCapitalLetters")
	var gibberishName = string("l337n@me")
	var goodName = string("good-name-for-cluster")
	t.Parallel()
	tests := []struct {
		name        string
		clusterName string
		want        bool
	}{
		// Negative tests for bad names. Should return false.
		{"Fails when too long", bigLongName, false},
		{"Fails with capital letters", capitalName, false},
		{"Fails with gibberish name", gibberishName, false},
		// Positive tests with good names. Should return true.
		{"Passes with good name", goodName, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClusterName(tt.clusterName)
			nameIsValid := false
			if err == nil {
				nameIsValid = true
			}
			assert.Equal(t, nameIsValid, tt.want)
		})
	}
}

func TestVerifyDomainName(t *testing.T) {
	t.Parallel()
	invalidErr := "domain name %s contains invalid characters"
	lengthErr := "domain name %s has fewer than 3 or greater than 63 characters"

	domain := "wine.com"
	assert.Equal(t, validateDomainName(domain), nil)
	domain = "more-wine.com"
	assert.Equal(t, validateDomainName(domain), nil)
	domain = "wine-and-cheese.com"
	assert.Equal(t, validateDomainName(domain), nil)
	domain = "wine-and-cheese.tasting.com"
	assert.Equal(t, validateDomainName(domain), nil)
	domain = "wine123.com"
	assert.Equal(t, validateDomainName(domain), nil)
	domain = "wine.cheese.com"
	assert.Equal(t, validateDomainName(domain), nil)
	domain = "win_e.com"
	assert.Equal(t, validateDomainName(domain), nil)

	domain = "win?e.com"
	assert.EqualError(t, validateDomainName(domain), fmt.Sprintf(invalidErr, domain))
	domain = "win%e.com"
	assert.EqualError(t, validateDomainName(domain), fmt.Sprintf(invalidErr, domain))
	domain = "om"

	assert.EqualError(t, validateDomainName(domain), fmt.Sprintf(lengthErr, domain))
	domain = "some.really.long.domain.that.should.be.longer.than.the.maximum.63.characters.com"
	assert.EqualError(t, validateDomainName(domain), fmt.Sprintf(lengthErr, domain))
}

func TestStripTrailingSlash(t *testing.T) {
	t.Parallel()

	url := "http://some.url.com/"
	assert.Equal(t, stripTrailingSlash(url), "http://some.url.com")

	url = "http://some.other.url.com"
	assert.Equal(t, stripTrailingSlash(url), "http://some.other.url.com")
}
