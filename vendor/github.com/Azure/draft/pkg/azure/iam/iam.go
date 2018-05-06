// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package iam

import (
	"errors"
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/cli"

	"github.com/Azure/draft/pkg/azure/iam/helpers"
)

// OAuthGrantType specifies which grant type to use.
type OAuthGrantType int

const (
	// OAuthGrantTypeServicePrincipal for client credentials flow
	OAuthGrantTypeServicePrincipal OAuthGrantType = iota
	// OAuthGrantTypeDeviceFlow for device-auth flow
	OAuthGrantTypeDeviceFlow
)

// AuthGrantType returns what kind of authentication is going to be used: device flow or service principal
func AuthGrantType() OAuthGrantType {
	if helpers.DeviceFlow() {
		return OAuthGrantTypeDeviceFlow
	}
	return OAuthGrantTypeServicePrincipal
}

// GetResourceManagementAuthorizer gets an OAuth token for managing resources using the specified grant type.
func GetResourceManagementAuthorizer(grantType OAuthGrantType) (autorest.Authorizer, error) {
	switch grantType {
	case OAuthGrantTypeServicePrincipal:
		tokenPath, err := cli.AccessTokensPath()
		if err != nil {
			return nil, fmt.Errorf("There was an error while grabbing the access token path: %v", err)
		}
		tokens, err := cli.LoadTokens(tokenPath)
		if err != nil {
			return nil, fmt.Errorf("There was an error loading the tokens from %s: %v", tokenPath, err)
		}
		for _, token := range tokens {
			adalToken, err := token.ToADALToken()
			if err != nil {
				continue
			}
			if adalToken.IsExpired() {
				continue
			}
			return autorest.NewBearerAuthorizer(&adalToken), nil
		}
		return nil, fmt.Errorf("run `az login` to get started")
	default:
		return nil, errors.New("invalid token type specified")
	}
}

// GetToken gets an OAuth token for managing resources using the specified grant type.
func GetToken(grantType OAuthGrantType) (adal.Token, error) {
	if grantType != OAuthGrantTypeServicePrincipal {
		return adal.Token{}, fmt.Errorf("unsupported grant type '%v'", grantType)
	}
	tokenPath, err := cli.AccessTokensPath()
	if err != nil {
		return adal.Token{}, fmt.Errorf("There was an error while grabbing the access token path: %v", err)
	}
	tokens, err := cli.LoadTokens(tokenPath)
	if err != nil {
		return adal.Token{}, fmt.Errorf("There was an error loading the tokens from %s: %v", tokenPath, err)
	}
	for _, token := range tokens {
		adalToken, err := token.ToADALToken()
		if err != nil {
			continue
		}
		if adalToken.IsExpired() {
			continue
		}
		return adalToken, nil
	}
	return adal.Token{}, fmt.Errorf("run `az login` to get started")

}
