// Copyright 2017 Codeship. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

/*
Package codeship provides a client for using the Codeship API v2.

The Codeship API v2 documentation exists at: https://apidocs.codeship.com/v2

Usage:

    import codeship "github.com/codeship/codeship-go"

Create a new API Client:

    auth := codeship.NewBasicAuth("username", "password")
    client, err := codeship.New(auth)

You must then scope the client to a single Organization that you have access to:

    org, err := client.Organization(ctx, "codeship")

You can then perform calls to the API on behalf of an Organization:

    projects, err := org.ListProjects(ctx)

*/
package codeship
